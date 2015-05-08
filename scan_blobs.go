// Copyright 2015 Aaron Jacobs. All Rights Reserved.
// Author: aaronjjacobs@gmail.com (Aaron Jacobs)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A command that processes all of the blobs in a GCS bucket. For each blob:
//
//  1. The GCS object record is verified for internal consistency. In
//     particular, comeback's expected MD5 and CRC32C are checked against the
//     GCS-reported values.
//
//  2. The object contents are read and the SHA-1 computed and checked against
//     the blob's name and expected SHA-1.
//
//  3. The edges in the directory graph are printed to an output file.
//
// (In "fast mode", only #1 is performed, and the output is a simple list of
// scores, one per line.)
//
// Output is space-separated lines of the form:
//
//     <score> <child score>*
//
// where <score> is a verified score, and the list is a list of scores of
// children for directories (and empty for files).

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var cmdScanBlobs = &Command{
	Name: "scan_blobs",
}

var fOutputFile = cmdScanBlobs.Flags.String(
	"output_file",
	"",
	"Path to scan_blobs output file. Will be truncated.")

var fFast = cmdScanBlobs.Flags.Bool(
	"fast",
	false,
	"Only verify object metadata.")

func init() {
	cmdScanBlobs.Run = runScanBlobs // Break flag-related dependency loop.
}

type scoreAndContents struct {
	score    blob.Score
	contents []byte
}

type verifiedScore struct {
	score    blob.Score
	children []blob.Score
}

// List all blob objects in the GCS bucket, and verify their metadata. Write
// their scores into the supplied channel.
func listBlobs(
	ctx context.Context,
	bucket gcs.Bucket,
	scores chan<- blob.Score) (err error) {
	b := syncutil.NewBundle()

	// List object records into a channel.
	objects := make(chan *gcs.Object, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(objects)
		err = blob.ListBlobObjects(ctx, bucket, blobObjectNamePrefix, objects)
		if err != nil {
			err = fmt.Errorf("ListBlobObjects: %v", err)
			return
		}

		return
	})

	// Parse and verify records, and write out scores.
	b.Add(func(ctx context.Context) (err error) {
		for o := range objects {
			// Parse and verify.
			var score Score
			score, err = ParseObjectRecord(o, s.namePrefix)
			if err != nil {
				err = fmt.Errorf("ParseObjectRecord: %v", err)
				return
			}

			// Send on the score.
			select {
			case scores <- score:

			// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		return
	})
}

func readAndClose(rc io.ReadCloser) (d []byte, err error) {
	// Read.
	d, err = ioutil.ReadAll(rc)
	if err != nil {
		rc.Close()
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}

	// Close.
	err = rc.Close()
	if err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

func readBlob(
	ctx context.Context,
	bucket gcs.Bucket,
	score blob.Score) (blob scoreAndContents, err error) {
	// Fill in the score.
	blob.score = score

	// Obtain a reader.
	req := &gcs.ReadObjectRequest{
		Name: blobObjectNamePrefix + score.Hex(),
	}

	rc, err := bucket.NewReader(ctx, req)
	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	// Consume it.
	blob.contents, err = readAndClose(rc)
	if err != nil {
		err = fmt.Errorf("readAndClose: %v", err)
		return
	}

	return
}

func readBlobWithRetryLoop(
	ctx context.Context,
	bucket gcs.Bucket,
	score blob.Score) (blob scoreAndContents, err error) {
	const maxRetries = 5
	for n := 0; n < maxRetries; n++ {
		blob, err = readBlob(ctx, bucket, score)
		if err == nil {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	return
}

// Read the contents of blobs specified on the incoming channel. Do not close
// the outgoing channel.
func readBlobs(
	ctx context.Context,
	bucket gcs.Bucket,
	scores <-chan blob.Score,
	blobs chan<- scoreAndContents) (err error) {
	for score := range scores {
		var blob scoreAndContents

		// Read the contents for this score.
		blob, err = readBlobWithRetryLoop(ctx, bucket, score)
		if err != nil {
			err = fmt.Errorf("readBlob: %v", err)
			return
		}

		select {
		case blobs <- blob:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

func scoresEqual(a, b blob.Score) bool {
	for i := 0; i < blob.ScoreLength; i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Parse the blob ciphertext as a directory if appropriate, returning a list of
// children. If not a directory, return the empty list.
func unmarshalBlob(
	crypter crypto.Crypter,
	b []byte) (children []blob.Score, err error) {
	// Attempt to decrypt.
	b, err = crypter.Decrypt(b)
	if err != nil {
		err = fmt.Errorf("Decrypt: %v", err)
		return
	}

	// If this is a file, simply make sure it is in a legal format.
	if !repr.IsDir(b) {
		_, err = repr.UnmarshalFile(b)
		if err != nil {
			err = fmt.Errorf("UnmarshalFile: %v", err)
			return
		}

		return
	}

	// Otherwise, parse it as a directory.
	entries, err := repr.UnmarshalDir(b)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir: %v", err)
		return
	}

	// Pull out the child scores.
	for _, entry := range entries {
		for _, score := range entry.Scores {
			children = append(children, score)
		}
	}

	return
}

// Verify the contents of the incoming blobs. Do not close the outgoing
// channel.
func verifyScores(
	ctx context.Context,
	crypter crypto.Crypter,
	blobs <-chan scoreAndContents,
	results chan<- verifiedScore) (err error) {
	for b := range blobs {
		result := verifiedScore{
			score: b.score,
		}

		// Make sure the score matches.
		computed := blob.ComputeScore(b.contents)
		if !scoresEqual(b.score, computed) {
			err = fmt.Errorf(
				"Score mismatch: %v vs. %v",
				b.score.Hex(),
				computed.Hex())

			return
		}

		// Parse the blob.
		result.children, err = unmarshalBlob(crypter, b.contents)
		if err != nil {
			err = fmt.Errorf("unmarshalBlob: %v", err)
			return
		}

		// Write out the result.
		select {
		case results <- result:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

// Write results to stdout.
func writeResults(
	ctx context.Context,
	output io.Writer,
	results <-chan verifiedScore) (err error) {
	// Log also to stdout.
	output = io.MultiWriter(output, os.Stdout)

	// Process each result.
	for result := range results {
		// Write the score.
		_, err = fmt.Fprintf(output, "%s", result.score.Hex())
		if err != nil {
			return
		}

		// Write its children.
		for _, child := range result.children {
			_, err = fmt.Fprintf(output, " %s", child.Hex())
			if err != nil {
				return
			}
		}

		// Add a line separator.
		_, err = fmt.Fprintf(output, "\n")
		if err != nil {
			return
		}
	}

	return
}

func runScanBlobs(args []string) {
	// Die on error.
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Open the output file.
	if *fOutputFile == "" {
		err = fmt.Errorf("You must set --output_file.")
		return
	}

	output, err := os.OpenFile(
		*fOutputFile,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666)

	if err != nil {
		err = fmt.Errorf("OpenFile: %v", err)
		return
	}

	defer output.Close()

	// Grab dependencies.
	bucket := getBucket()
	var crypter crypto.Crypter
	if !*fFast {
		crypter = getCrypter()
	}

	// Allow parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	b := syncutil.NewBundle(context.Background())

	// List all of the scores in the bucket.
	scores := make(chan blob.Score, 5000)
	b.Add(func(ctx context.Context) (err error) {
		defer close(scores)
		err = listBlobs(ctx, bucket, scores)
		return
	})

	results := make(chan verifiedScore, 100)
	if !*fFast {
		// Read the contents of the corresponding blobs in parallel, bounding how
		// hard we hammer GCS by bounding the parallelism.
		const readWorkers = 8
		var readWaitGroup sync.WaitGroup

		blobs := make(chan scoreAndContents, 10)
		for i := 0; i < readWorkers; i++ {
			readWaitGroup.Add(1)
			b.Add(func(ctx context.Context) (err error) {
				defer readWaitGroup.Done()
				err = readBlobs(ctx, bucket, scores, blobs)
				return
			})
		}

		go func() {
			readWaitGroup.Wait()
			close(blobs)
		}()

		// Verify the blob contents and summarize their children. Use one worker per
		// CPU.
		var verifyWaitGroup sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			verifyWaitGroup.Add(1)
			b.Add(func(ctx context.Context) (err error) {
				defer verifyWaitGroup.Done()
				err = verifyScores(ctx, crypter, blobs, results)
				return
			})
		}

		go func() {
			verifyWaitGroup.Wait()
			close(results)
		}()
	} else {
		// Fast mode. Pretend that everything is verified, and that nothing has
		// children.
		b.Add(func(ctx context.Context) (err error) {
			defer close(results)
			for score := range scores {
				select {
				case results <- verifiedScore{score: score}:

				// Cancelled?
				case <-ctx.Done():
					err = ctx.Err()
					return
				}
			}

			return
		})
	}

	// Process results.
	b.Add(func(ctx context.Context) (err error) {
		err = writeResults(ctx, output, results)
		return
	})

	// Wait for everything to complete.
	err = b.Join()
	return
}

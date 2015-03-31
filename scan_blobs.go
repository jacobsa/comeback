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

// A command that scans all of the blobs in a GCS bucket, verifying their
// scores against their content.
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
	"runtime"
	"strings"
	"sync"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var cmdScanBlobs = &Command{
	Name: "scan_blobs",
	Run:  runScanBlobs,
}

type scoreAndContents struct {
	score    blob.Score
	contents []byte
}

type verifiedScore struct {
	score    blob.Score
	children []blob.Score
}

// List all scores in the GCS bucket into the channel. Do not close the
// channel.
func listBlobs(
	ctx context.Context,
	bucket gcs.Bucket,
	scores chan<- blob.Score) (err error) {
	req := &gcs.ListObjectsRequest{
		Prefix: blobKeyPrefix,
	}

	// List until we run out.
	for {
		// Fetch the next batch.
		var listing *gcs.Listing
		listing, err = bucket.ListObjects(ctx, req)
		if err != nil {
			err = fmt.Errorf("ListObjects: %v", err)
			return
		}

		// Transform to scores.
		var batch []blob.Score
		for _, o := range listing.Objects {
			var score blob.Score
			score, err = blob.ParseHexScore(strings.TrimPrefix(o.Name, blobKeyPrefix))
			if err != nil {
				err = fmt.Errorf("Parsing object name \"%s\": %v", o.Name, err)
				return
			}

			batch = append(batch, score)
		}

		// Feed out each score.
		for _, score := range batch {
			select {
			case scores <- score:

				// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		// Are we done?
		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	return
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
		Name: blobKeyPrefix + score.Hex(),
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
		blob, err = readBlob(ctx, bucket, score)
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
	results <-chan verifiedScore) (err error) {
	for result := range results {
		// Write the score.
		fmt.Printf("%s", result.score.Hex())

		// Write its children.
		for _, child := range result.children {
			fmt.Printf(" %s", child.Hex())
		}

		// Add a line separator.
		fmt.Printf("\n")
	}

	return
}

func runScanBlobs(args []string) {
	var err error
	b := syncutil.NewBundle(context.Background())

	// Grab dependencies.
	bucket := getBucket()
	crypter := getCrypter()

	// Die on error.
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Allow parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// List all of the scores in the bucket.
	scores := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(scores)
		err = listBlobs(ctx, bucket, scores)
		return
	})

	// Read the contents of the corresponding blobs in parallel, bounding how
	// hard we hammer GCS by bounding the parallelism.
	const readWorkers = 128
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

	results := make(chan verifiedScore, 100)
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

	// Process results.
	b.Add(func(ctx context.Context) (err error) {
		err = writeResults(ctx, results)
		return
	})

	// Wait for everything to complete.
	err = b.Join()
	return
}

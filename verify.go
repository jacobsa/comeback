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

// A command that reads all blobs necessary for verifying the directory
// structure rooted at a set of backup root scores, ensuring that the entire
// directory structure is intact in GCS.
//
// Optionally, all file content is also read and verified. This is less
// important than verifying directory connectedness if we trust that GCS does
// not corrupt object metadata (where we store expected CRC32C and MD5) and
// does correctly report the object's CRC32C and MD5 sums in listings,
// verifying them periodically.
//
// Output of the following form is written to stdout and a file in the user's
// home directory:
//
//     <timestamp> <node> [<child node> ...]
//
// where:
//
//  *  Timestamps are formatted according to time.RFC3339.
//
//  *  Node names have one of two forms:
//
//     *   Nodes of the form "d:<hex score>" represent the directory listing
//         contained within the blob of the given score.
//
//     *   Nodes of the form "f:<hex score>" represent a piece of a file,
//         contained within the blob of the given score.
//
// An output line for a directory node means that at the given timestamp we
// certified that a piece of content with the given score was parseable as a
// directory listing that referred to the given scores for its direct children.
//
// An output line for a file node means that at the given timestamp we
// certified that a piece of content with the given score was parseable as a
// piece of a file. File nodes never have children.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/verify"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/gcloud/gcs"
)

var cmdVerify = &Command{
	Name: "verify",
}

var fFast = cmdVerify.Flags.Bool(
	"fast",
	false,
	"When set, don't verify file content.")

func init() {
	cmdVerify.Run = runVerify // Break flag-related dependency loop.
}

////////////////////////////////////////////////////////////////////////
// Output
////////////////////////////////////////////////////////////////////////

// Parse the supplied output from a run of the verify command, writing records
// into the supplied channel.
func parseVerifyOutput(
	ctx context.Context,
	in io.Reader,
	records chan<- verify.Record) (err error) {
	reader := bufio.NewReader(in)

	for {
		// Find the next line. EOF with no data means we are done; otherwise ignore
		// EOF in case the file doesn't end with a newline.
		var line []byte
		line, err = reader.ReadBytes('\n')
		if err == io.EOF {
			err = nil
			if len(line) == 0 {
				break
			}
		}

		// Propagate other errors.
		if err != nil {
			err = fmt.Errorf("ReadBytes: %v", err)
			return
		}

		// Trim the delimiter.
		line = line[:len(line)-1]

		// Parse the line.
		var r verify.Record
		r, err = verify.ParseRecord(string(line))
		if err != nil {
			err = fmt.Errorf("parseVerifyRecord(%q): %v", line, err)
			return
		}

		// Write out the record.
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return

		case records <- r:
		}
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// List blob.ListScores, but returns a slice instead of writing into a channel.
func listAllScores(
	ctx context.Context,
	bucket gcs.Bucket,
	namePrefix string) (scores []blob.Score, err error) {
	eg, ctx := errgroup.WithContext(ctx)
	defer func() { err = eg.Wait() }()

	// List scores into a channel.
	scoreChan := make(chan blob.Score, 100)
	eg.Go(func() (err error) {
		defer close(scoreChan)
		err = blob.ListScores(ctx, bucket, wiring.BlobObjectNamePrefix, scoreChan)
		if err != nil {
			err = fmt.Errorf("ListScores: %v", err)
			return
		}

		return
	})

	// Accumulate into the slice.
	eg.Go(func() (err error) {
		for score := range scoreChan {
			scores = append(scores, score)
		}

		return
	})

	return
}

////////////////////////////////////////////////////////////////////////
// Verify
////////////////////////////////////////////////////////////////////////

// Run the verification pipeline. Return a count of the number of scores
// verified.
func verifyImpl(
	ctx context.Context,
	readFiles bool,
	rootScores []blob.Score,
	knownScores []blob.Score,
	knownStructure map[verify.Node][]verify.Node,
	blobStore blob.Store,
	output io.Writer) (nodesVerified uint64, err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// Call the workhorse function, writing records int o a channel.
	records := make(chan verify.Record, 100)
	eg.Go(func() (err error) {
		defer close(records)
		err = verify.Verify(
			ctx,
			readFiles,
			rootScores,
			knownScores,
			knownStructure,
			records,
			blobStore)

		if err != nil {
			err = fmt.Errorf("verify.Verify: %v", err)
			return
		}

		return
	})

	// Count and output the nodes visited.
	eg.Go(func() (err error) {
		for r := range records {
			// Increment the count.
			nodesVerified++

			// Output the information.
			_, err = fmt.Fprintf(output, "%s\n", r.String())
			if err != nil {
				err = fmt.Errorf("Fprintf: %v", err)
				return
			}
		}

		return
	})

	err = eg.Wait()
	return
}

// Parse the contents of the verify log into a known structure map as accepted
// by verify.NewVisitor.
func parseKnownStructure(
	ctx context.Context,
	r io.Reader) (knownStructure map[verify.Node][]verify.Node, err error) {
	const stalenessThreshold = 90 * 24 * time.Hour
	eg, ctx := errgroup.WithContext(ctx)

	// Parse records into a channel.
	records := make(chan verify.Record, 100)
	eg.Go(func() (err error) {
		defer close(records)
		err = parseVerifyOutput(ctx, r, records)
		if err != nil {
			err = fmt.Errorf("parseVerifyOutput: %v", err)
			return
		}

		return
	})

	// Accumulate into the map.
	knownStructure = make(map[verify.Node][]verify.Node)
	eg.Go(func() (err error) {
		now := time.Now()
		for r := range records {
			// Skip stale records.
			if now.Sub(r.Time) > stalenessThreshold {
				continue
			}

			knownStructure[r.Node] = r.Children
		}

		return
	})

	err = eg.Wait()
	return
}

// Open the file to which we log verify output. Read its current contents,
// filtering out entries that are too old, and return a writer that can be used
// to append to it.
func openVerifyLog(ctx context.Context) (
	w io.WriteCloser,
	knownStructure map[verify.Node][]verify.Node,
	err error) {
	// Find the current user.
	u, err := user.Current()
	if err != nil {
		err = fmt.Errorf("user.Current: %v", err)
		return
	}

	// Put the file in her home directory. Append to whatever is already there.
	f, err := os.OpenFile(
		path.Join(u.HomeDir, ".comeback.verify.log"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	if err != nil {
		err = fmt.Errorf("OpenFile: %v", err)
		return
	}

	// Seek to the beginning and parse.
	_, err = f.Seek(0, 0)
	if err != nil {
		f.Close()
		err = fmt.Errorf("Seek: %v", err)
		return
	}

	knownStructure, err = parseKnownStructure(ctx, f)
	if err != nil {
		f.Close()
		err = fmt.Errorf("parseVerifyLog: %v", err)
		return
	}

	w = f
	return
}

func runVerify(ctx context.Context, args []string) (err error) {
	readFiles := !*fFast

	// Grab dependencies.
	bucket := getBucket(ctx)
	registry := getRegistry(ctx)
	blobStore := getBlobStore(ctx)

	// Open the log file.
	logFile, knownStructure, err := openVerifyLog(ctx)
	if err != nil {
		err = fmt.Errorf("openVerifyLog: %v", err)
		return
	}

	defer logFile.Close()

	// Find the root scores to be verified.
	jobs, err := registry.ListBackups(ctx)
	if err != nil {
		err = fmt.Errorf("ListBackups: %v", err)
		return
	}

	var rootScores []blob.Score
	log.Println("Root scores to be verified:")
	for _, j := range jobs {
		rootScores = append(rootScores, j.Score)
		log.Printf("  %s", j.Score.Hex())
	}

	// List all scores in the bucket, verifying the object record metadata in the
	// process.
	log.Println("Listing scores...")
	knownScores, err := listAllScores(
		ctx,
		bucket,
		wiring.BlobObjectNamePrefix)

	if err != nil {
		err = fmt.Errorf("listAllScores: %v", err)
		return
	}

	log.Printf("Listed %d scores.", len(knownScores))

	// Run the rest of the pipeline.
	nodesVerified, err := verifyImpl(
		ctx,
		readFiles,
		rootScores,
		knownScores,
		knownStructure,
		blobStore,
		io.MultiWriter(logFile, os.Stdout))

	if err != nil {
		err = fmt.Errorf("verifyImpl: %v", err)
		return
	}

	log.Printf("Successfully verified %d nodes.", nodesVerified)
	return
}

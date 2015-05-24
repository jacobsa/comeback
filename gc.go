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

// A command that consumes the output of a `comeback verify` run (--fast mode
// is okay), assuming that the roots were all backup jobs of interest. Any
// score that is in the bucket but not represented in the verify output is
// cloned to a garbage/ prefix in the bucket, and deleted from the blobs/
// prefix.

package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/wiring"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var cmdGC = &Command{
	Name: "gc",
}

var fInput = cmdGC.Flags.String(
	"input",
	"",
	"Path to a file containing the output of a verify run.")

func init() {
	cmdGC.Run = runGC // Break flag-related dependency loop.
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Parse the verify output, returning a list of all scores encountered.
func parseInput(
	r io.Reader) (scores []blob.Score, err error) {
	err = errors.New("TODO: parseInput")
	return
}

// Filter out scores that are in the list of non-garbage accessible scores,
// passing on only garbage.
func filterToGarbage(
	ctx context.Context,
	accessible []blob.Score,
	allScores <-chan blob.Score,
	garbageScores chan<- blob.Score) (err error) {
	err = errors.New("TODO: filterToGarbage")
	return
}

// Count the scores that pass through.
func countScores(
	ctx context.Context,
	in <-chan blob.Score,
	out chan<- blob.Score) (count uint64, err error) {
	err = errors.New("TODO: countScores")
	return
}

// Clone garbage blobs into a new location. Pass on the names of the source
// objects that were cloned.
func cloneGarbage(
	ctx context.Context,
	bucket gcs.Bucket,
	garbageScores <-chan blob.Score,
	garbageObjects chan<- string) (err error) {
	err = errors.New("TODO: cloneGarbage")
	return
}

// Delete all objects whose name come in on the supplied channel.
func deleteObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	names <-chan string) (err error) {
	err = errors.New("TODO: deleteObjects")
	return
}

////////////////////////////////////////////////////////////////////////
// GC
////////////////////////////////////////////////////////////////////////

func runGC(args []string) {
	// Allow parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Die on error.
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Grab dependencies.
	bucket := getBucket()

	// Open the input file.
	if *fInput == "" {
		err = fmt.Errorf("You must set --input.")
		return
	}

	inputFile, err := os.Open(*fInput)
	if err != nil {
		err = fmt.Errorf("Open: %v", err)
		return
	}

	// Parse it.
	accessibleScores, err := parseInput(inputFile)
	inputFile.Close()

	if err != nil {
		err = fmt.Errorf("parseInput: %v", err)
		return
	}

	b := syncutil.NewBundle(context.Background())

	// List all extant scores into a channel.
	allScores := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(allScores)
		err = blob.ListScores(ctx, bucket, wiring.BlobObjectNamePrefix, allScores)
		if err != nil {
			err = fmt.Errorf("ListScores: %v", err)
			return
		}

		return
	})

	// Count the total number of scores.
	var allScoresCount uint64
	allScoresAfterCounting := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(allScoresAfterCounting)
		allScoresCount, err = countScores(
			ctx,
			allScores,
			allScoresAfterCounting)

		if err != nil {
			err = fmt.Errorf("countScores: %v", err)
			return
		}

		return
	})

	// Filter to garbage scores.
	garbageScores := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(garbageScores)
		err = filterToGarbage(
			ctx,
			accessibleScores,
			allScoresAfterCounting,
			garbageScores)

		if err != nil {
			err = fmt.Errorf("filterToGarbage: %v", err)
			return
		}

		return
	})

	// Count the number of garbage scores.
	var garbageScoresCount uint64
	garbageScoresAfterCounting := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(garbageScoresAfterCounting)
		garbageScoresCount, err = countScores(
			ctx,
			garbageScores,
			garbageScoresAfterCounting)

		if err != nil {
			err = fmt.Errorf("countScores: %v", err)
			return
		}

		return
	})

	// Clone garbage blobs into a backup location.
	toDelete := make(chan string, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(toDelete)
		err = cloneGarbage(
			ctx,
			bucket,
			garbageScoresAfterCounting,
			toDelete)
		if err != nil {
			err = fmt.Errorf("cloneGarbage: %v", err)
			return
		}

		return
	})

	// Delete the original objects.
	b.Add(func(ctx context.Context) (err error) {
		err = deleteObjects(ctx, bucket, toDelete)
		if err != nil {
			err = fmt.Errorf("deleteObjects: %v", err)
			return
		}

		return
	})

	err = b.Join()
	if err != nil {
		return
	}

	// Print a summary.
	log.Printf(
		"Deleted %d objects out of %d total.",
		garbageScoresCount,
		allScoresCount)
}

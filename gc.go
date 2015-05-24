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
	"io"
	"log"
	"runtime"

	"github.com/jacobsa/comeback/blob"
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
func parseInput(r io.Reader) (scores []blob.Score, err error)

// Filter out scores that are in the list of non-garbage accessible scores,
// passing on only garbage.
func filterToGarbage(
	ctx context.Context,
	accessible []blob.Score,
	allScores <-chan blob.Score,
	garbageScores chan<- blob.Score) (err error)

// Count the scores that pass through.
func countScores(
	ctx context.Context,
	in <-chan blob.Score,
	out chan<- blob.Score) (count uint64, err error)

// Clone garbage blobs into a new location. Pass on the names of the source
// objects that were cloned.
func cloneGarbage(
	ctx context.Context,
	garbageScores <-chan blob.Score,
	garbageObjects chan<- string) (err error)

// Delete all objects whose name come in on the supplied channel.
func deleteObjects(
	ctx context.Context,
	names <-chan string) (err error)

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

	err = errors.New("TODO: runGC")
}

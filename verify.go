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

package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/util"
	"github.com/jacobsa/comeback/verify"
	"github.com/jacobsa/comeback/wiring"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var cmdVerify = &Command{
	Name: "verify",
}

// TODO(jacobsa): Get these automatically from the registry.
var fRoots = cmdVerify.Flags.String(
	"roots",
	"",
	"Comma-separated list of backup root scores to verify.")

var fFast = cmdVerify.Flags.Bool(
	"fast",
	false,
	"When set, don't verify file content.")

func init() {
	cmdVerify.Run = runVerify // Break flag-related dependency loop.
}

// List blob.ListScores, but returns a slice instead of writing into a channel.
func listAllScores(
	ctx context.Context,
	bucket gcs.Bucket,
	namePrefix string) (scores []blob.Score, err error) {
	b := syncutil.NewBundle(context.Background())
	defer func() { err = b.Join() }()

	// List scores into a channel.
	scoreChan := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(scoreChan)
		err = blob.ListScores(ctx, bucket, wiring.BlobObjectNamePrefix, scoreChan)
		if err != nil {
			err = fmt.Errorf("ListScores: %v", err)
			return
		}

		return
	})

	// Accumulate into the slice.
	b.Add(func(ctx context.Context) (err error) {
		for score := range scoreChan {
			scores = append(scores, score)
		}

		return
	})

	return
}

func runVerify(args []string) {
	// Allow parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Die on error.
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Read flags.
	readFiles := !*fFast

	if *fRoots == "" {
		err = fmt.Errorf("You must set --roots.")
		return
	}

	rootHexScores := strings.Split(*fRoots, ",")
	var roots []string
	for _, hexScore := range rootHexScores {
		var score blob.Score
		score, err = blob.ParseHexScore(hexScore)
		if err != nil {
			err = fmt.Errorf("Invalid root %q: %v", hexScore, err)
			return
		}

		roots = append(roots, verify.FormatNodeName(true, score))
	}

	// Grab dependencies.
	bucket := getBucket()
	crypter := getCrypter()

	// Create a blob store.
	blobStore, err := wiring.MakeBlobStore(
		bucket,
		crypter,
		util.NewStringSet())

	if err != nil {
		err = fmt.Errorf("MakeBlobStore: %v", err)
		return
	}

	// List all scores in the bucket, verifying the object record metadata in the
	// process.
	log.Println("Listing scores...")
	knownScores, err := listAllScores(
		context.Background(),
		bucket,
		wiring.BlobObjectNamePrefix)

	if err != nil {
		err = fmt.Errorf("listAllScores: %v", err)
		return
	}

	log.Printf("Listed %d scores.", len(knownScores))

	// Create a graph visitor that perform the verification.
	visitor := verify.NewVisitor(
		readFiles,
		knownScores,
		blobStore)

	// Traverse starting at the specified roots. Use an "experimentally
	// determined" parallelism, which in theory should depend on bandwidth-delay
	// products but in practice comes down to when the OS gets cranky about open
	// files.
	const parallelism = 256

	err = graph.Traverse(
		context.Background(),
		parallelism,
		roots,
		visitor)

	if err != nil {
		err = fmt.Errorf("Traverse: %v", err)
		return
	}

	return
}

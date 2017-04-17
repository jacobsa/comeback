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

package save

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/crypto"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/timeutil"
)

// Save a backup of the given directory, applying the supplied exclusions and
// using the supplied score map to avoid reading file content when possible.
// Return a score for the root of the backup.
//
// The supplied bucket will be used to store objects with the given name
// prefix. existingScores must contain only scores that are known to exist in
// the bucket, in hex form. It will be updated as blobs are saved to the
// bucket.
func Save(
	ctx context.Context,
	dir string,
	exclusions []*regexp.Regexp,
	bucket gcs.Bucket,
	objectNamePrefix string,
	crypter crypto.Crypter,
	existingScores util.StringSet,
	scoreMap state.ScoreMap,
	logger *log.Logger,
	clock timeutil.Clock) (score blob.Score, err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// Visit each node in the graph, writing the processed nodes to a channel.
	processedNodes := make(chan *fsNode, 100)
	eg.Go(func() (err error) {
		defer close(processedNodes)

		// The resolver only makes use of the local file system. It actually seems
		// to hurt to have parallelism here, presumably because it ruins locality
		// in what otherwise would be LIFO processing of file system nodes.
		const resolverParallelism = 1

		// The visitor reads contents, computes SHA-1s, encrypts, and talks to GCS.
		// Hopefully this is enough parallelism to keep our CPUs or NIC saturated,
		// depending on which is the current bottleneck.
		const visitorParallelism = 128

		visitor := newVisitor(
			fileChunkSize,
			dir,
			scoreMap,
			newBlobStore(bucket, objectNamePrefix, crypter, existingScores),
			clock,
			logger,
			processedNodes)

		err = dag.Visit(
			ctx,
			[]dag.Node{makeRootNode()},
			newDependencyResolver(dir, exclusions),
			visitor,
			resolverParallelism,
			visitorParallelism)

		if err != nil {
			err = fmt.Errorf("dag.Visit: %v", err)
			return
		}

		return
	})

	// Find the root score.
	eg.Go(func() (err error) {
		score, err = findRootScore(processedNodes)
		if err != nil {
			err = fmt.Errorf("findRootScore: %v", err)
			return
		}

		return
	})

	err = eg.Wait()
	return
}

// newBlobStore creates a blob store that stores blobs in the supplied bucket
// under the given name prefix, encrypting with the supplied crypter.
//
// existingScores must contain only scores that are known to exist in the
// bucket, in hex form. It will be updated as the blob store is used.
func newBlobStore(
	bucket gcs.Bucket,
	objectNamePrefix string,
	crypter crypto.Crypter,
	existingScores util.StringSet) (bs blob.Store) {
	// Store blobs in GCS.
	bs = blob.NewGCSStore(bucket, objectNamePrefix)

	// Don't make redundant calls to GCS.
	bs = blob.NewExistingScoresStore(existingScores, bs)

	// Make paranoid checks on the results.
	bs = blob.NewCheckingStore(bs)

	// Encrypt blob data before sending it off to GCS.
	bs = blob.NewEncryptingStore(crypter, bs)

	return
}

func findRootScore(nodes <-chan *fsNode) (score blob.Score, err error) {
	found := false
	for n := range nodes {
		// Skip non-root nodes.
		if n.RelPath != "" {
			continue
		}

		// Is this a duplicate?
		if found {
			err = fmt.Errorf("Found a duplicate root node: %#v", n)
			return
		}

		found = true

		// We expect directory nodes to have exactly one score.
		if len(n.Info.Scores) != 1 {
			err = fmt.Errorf("Unexpected score count for rooT: %#v", n)
			return
		}

		score = n.Info.Scores[0]
	}

	if !found {
		err = errors.New("No root node found")
		return
	}

	return
}

// Create a node appropriate to pass as a start node to dag.Visit.
func makeRootNode() *fsNode {
	return &fsNode{
		RelPath: "",
		Info: fs.FileInfo{
			Type: fs.TypeDirectory,
		},
	}
}

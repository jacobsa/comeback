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
	"errors"
	"fmt"
	"log"
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
)

// Save a backup of the given directory, applying the supplied exclusions and
// using the supplied score map to avoid reading file content when possible.
// Return a score for the root of the backup.
func Save(
	ctx context.Context,
	dir string,
	exclusions []*regexp.Regexp,
	scoreMap state.ScoreMap,
	blobStore blob.Store,
	logger *log.Logger,
	clock timeutil.Clock) (score blob.Score, err error) {
	b := syncutil.NewBundle(ctx)

	// Visit each node in the graph, writing the processed nodes to a channel.
	processedNodes := make(chan *fsNode, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(processedNodes)

		// Hopefully enough parallelism to keep our CPUs saturated (for encryption,
		// SHA-1 computation, etc.) or our NIC saturated (for GCS traffic),
		// depending on which is the current bottleneck.
		const parallelism = 128

		visitor := newVisitor(
			fileChunkSize,
			dir,
			scoreMap,
			blobStore,
			clock,
			logger,
			processedNodes)

		err = dag.Visit(
			ctx,
			[]dag.Node{makeRootNode()},
			newDependencyResolver(dir, exclusions),
			visitor,
			parallelism)

		if err != nil {
			err = fmt.Errorf("dag.Visit: %v", err)
			return
		}

		return
	})

	// Find the root score.
	b.Add(func(ctx context.Context) (err error) {
		score, err = findRootScore(processedNodes)
		if err != nil {
			err = fmt.Errorf("findRootScore: %v", err)
			return
		}

		return
	})

	err = b.Join()
	return
}

func findRootScore(nodes <-chan *fsNode) (score blob.Score, err error) {
	found := false
	for n := range nodes {
		// Skip non-root nodes.
		if n.Parent != nil {
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
		Info: fs.DirectoryEntry{
			Type: fs.TypeDirectory,
		},
		Parent: nil,
	}
}

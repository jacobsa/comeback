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
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
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
	clock timeutil.Clock) (score blob.Score, err error) {
	b := syncutil.NewBundle(ctx)

	// List the directory hierarchy, applying exclusions.
	listedNodes := make(chan *fsNode, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(listedNodes)
		err = listNodes(ctx, dir, exclusions, listedNodes)
		if err != nil {
			err = fmt.Errorf("listNodes: %v", err)
			return
		}

		return
	})

	// Fill in scores for files that don't appear to have changed since the last
	// run.
	postScoreMap := make(chan *fsNode, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(postScoreMap)
		err = consultScoreMap(ctx, scoreMap, clock, listedNodes, postScoreMap)
		if err != nil {
			err = fmt.Errorf("consultScoreMap: %v", err)
			return
		}

		return
	})

	// Fill in scores for those nodes that didn't hit in the cache. Do this
	// safely, respecting dependency order (children complete before parents
	// start).
	postDAGTraversal := make(chan *fsNode, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(postDAGTraversal)
		err = fillInScores(ctx, dir, blobStore, postScoreMap, postDAGTraversal)
		if err != nil {
			err = fmt.Errorf("fillInScores: %v", err)
			return
		}

		return
	})

	// Update the score map with the results of the previous stage.
	postScoreMapUpdate := make(chan *fsNode, 100)
	{
		// Tee the channel; updateScoreMap doesn't give output, nor does it modify
		// nodes.
		tmp := make(chan *fsNode, 100)
		b.Add(func(ctx context.Context) (err error) {
			defer close(tmp)
			defer close(postScoreMapUpdate)

			err = teeNodes(ctx, postDAGTraversal, tmp, postScoreMapUpdate)
			return
		})

		// Run updateScoreMap.
		b.Add(func(ctx context.Context) (err error) {
			err = updateScoreMap(ctx, scoreMap, tmp)
			if err != nil {
				err = fmt.Errorf("updateScoreMap: %v", err)
				return
			}

			return
		})
	}

	// Find the root score.
	b.Add(func(ctx context.Context) (err error) {
		score, err = findRootScore(postScoreMapUpdate)
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

func teeNodes(
	ctx context.Context,
	in <-chan *fsNode,
	out1 chan<- *fsNode,
	out2 chan<- *fsNode) (err error) {
	for n := range in {
		// Write to first output.
		select {
		case out1 <- n:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}

		// And the second.
		select {
		case out2 <- n:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

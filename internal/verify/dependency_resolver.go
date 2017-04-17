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

package verify

import (
	"context"
	"fmt"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/timeutil"
)

// Create a dependency resolver for the DAG of blobs in the supplied bucket.
// Nodes are expected to be of type Node.
//
// For all nodes N, the resolver confirms that N.Score is in allScores.
// The resolver further does the following for directories:
//
//  *  If N is in knownStructure, return dependencies according to its entry
//     there.
//
//  *  Otherwise, load N.Score from the blob store, parse the listing, and
//     return appropriate dependencies. Write a record reflecting this to the
//     supplied channel.
//
// It is expected that the blob store's Load method does score verification for
// us.
func newDependencyResolver(
	allScores []blob.Score,
	knownStructure map[Node][]Node,
	records chan<- Record,
	bs blob.Store,
	clock timeutil.Clock) (dr dag.DependencyResolver) {
	scoreSet := make(map[blob.Score]struct{})
	for _, s := range allScores {
		scoreSet[s] = struct{}{}
	}

	dr = &dependencyResolver{
		allScores:      scoreSet,
		knownStructure: knownStructure,
		records:        records,
		blobStore:      bs,
		clock:          clock,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type dependencyResolver struct {
	allScores      map[blob.Score]struct{}
	knownStructure map[Node][]Node
	records        chan<- Record
	blobStore      blob.Store
	clock          timeutil.Clock
}

func (dr *dependencyResolver) FindDependencies(
	ctx context.Context,
	untyped dag.Node) (deps []dag.Node, err error) {
	// Make sure the node is of the appropriate type.
	n, ok := untyped.(Node)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	// Confirm that the score actually exists.
	if _, ok := dr.allScores[n.Score]; !ok {
		err = fmt.Errorf("Unknown score for node: %s", n.String())
		return
	}

	// There is nothing further to do unless this is a directory.
	if !n.Dir {
		return
	}

	// Do we already have the answer for this node?
	if children, ok := dr.knownStructure[n]; ok {
		for _, child := range children {
			deps = append(deps, child)
		}

		return
	}

	// Load the blob contents.
	contents, err := dr.blobStore.Load(ctx, n.Score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", n.Score.Hex(), err)
		return
	}

	// Parse the listing.
	listing, err := repr.UnmarshalDir(contents)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir(%s): %v", n.Score.Hex(), err)
		return
	}

	// Build a record containing a child node for each score in each entry.
	r := Record{
		Time: dr.clock.Now(),
		Node: n,
	}

	for _, entry := range listing {
		var child Node

		// Is this a directory?
		switch entry.Type {
		case fs.TypeFile:
			child.Dir = false

		case fs.TypeDirectory:
			child.Dir = true

		case fs.TypeSymlink:
			if len(entry.Scores) != 0 {
				err = fmt.Errorf(
					"Dir %s: symlink unexpectedly contains scores",
					n.Score.Hex())

				return
			}

		default:
			err = fmt.Errorf(
				"Dir %s: unknown entry type %v",
				n.Score.Hex(),
				entry.Type)

			return
		}

		// Add a node for each score.
		for _, score := range entry.Scores {
			child.Score = score
			r.Children = append(r.Children, child)
		}
	}

	// Certify that we verified the directory.
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case dr.records <- r:
	}

	// Copy the directory's children to our return value.
	for _, child := range r.Children {
		deps = append(deps, child)
	}

	return
}

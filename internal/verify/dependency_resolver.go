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
	"fmt"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"golang.org/x/net/context"
)

// Create a dependency resolver for the DAG of blobs in the supplied bucket.
// Nodes are expected to be of type Node.
//
// The resolver reads directory blobs, parses them, and returns their children
// as dependencies. (File nodes have no dependencies.) If work is to be
// preserved across runs, knownStructure should be filled in with parenthood
// information from previous verification runs.
//
// It is expected that the blob store's Load method does score verification for
// us.
func newDependencyResolver(
	knownStructure map[Node][]Node,
	bs blob.Store) (dr dag.DependencyResolver) {
	dr = &dependencyResolver{
		knownStructure: knownStructure,
		blobStore:      bs,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type dependencyResolver struct {
	knownStructure map[Node][]Node
	blobStore      blob.Store
}

func (sf *successorFinder) processFile(
	ctx context.Context,
	n Node) (err error) {
	// If reading files is disabled, there is nothing further to do.
	if !sf.readFiles {
		return
	}

	// Make sure we can load the blob contents. Presumably the blob store
	// verifies the score (of the ciphertext) on the way through.
	_, err = sf.blobStore.Load(ctx, n.Score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %sf", n.Score.Hex(), err)
		return
	}

	// Certify that we verified the file piece.
	r := Record{
		Time: sf.clock.Now(),
		Node: n,
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case sf.records <- r:
	}

	return
}

func (sf *successorFinder) processDir(
	ctx context.Context,
	parent Node) (successors []graph.Node, err error) {
	// Load the blob contents.
	contents, err := sf.blobStore.Load(ctx, parent.Score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", parent.Score.Hex(), err)
		return
	}

	// Parse the listing.
	listing, err := repr.UnmarshalDir(contents)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir(%s): %v", parent.Score.Hex(), err)
		return
	}

	// Build a record containing a child node for each score in each entry.
	r := Record{
		Time: sf.clock.Now(),
		Node: parent,
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
					parent.Score.Hex())

				return
			}

		default:
			err = fmt.Errorf(
				"Dir %s: unknown entry type %v",
				parent.Score.Hex(),
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

	case sf.records <- r:
	}

	// Copy the directory's children to our return value.
	for _, child := range r.Children {
		successors = append(successors, child)
	}

	return
}

func (sf *successorFinder) FindDirectSuccessors(
	ctx context.Context,
	untyped graph.Node) (successors []graph.Node, err error) {
	// Make sure the node is of the appropriate type.
	n, ok := untyped.(Node)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	// Make sure the score actually exists.
	if _, ok := sf.knownScores[n.Score]; !ok {
		err = fmt.Errorf("Unknown score for node: %s", n.String())
		return
	}

	// If we have already verified this node, there is nothing further to verify.
	// Return the appropriate successors.
	if children, ok := sf.knownStructure[n]; ok {
		for _, child := range children {
			successors = append(successors, child)
		}

		return
	}

	// Perform file or directory-specific logic.
	if n.Dir {
		successors, err = sf.processDir(ctx, n)
		return
	} else {
		err = sf.processFile(ctx, n)
		return
	}
}

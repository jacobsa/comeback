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

func (dr *dependencyResolver) FindDependencies(
	ctx context.Context,
	untyped dag.Node) (deps []dag.Node, err error) {
	// Make sure the node is of the appropriate type.
	n, ok := untyped.(Node)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
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

	// Fill in dependencies.
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
			deps = append(deps, child)
		}
	}

	return
}

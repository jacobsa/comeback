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
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/graph"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// Create a successor finder for the DAG of blobs in the supplied bucket. Nodes
// are expected to be of type Node.
//
// The successor finder reads directory blobs, parses them, and returns their
// children as adjacent nodes. For file nodes, the successor finder verifies
// that their score exists (according to allScores), and verifies that the blob
// can be loaded if readFiles is true. A record is written to the supplied
// channel for everything that is verified. This makes this SuccessorFinder a
// bit weird, in that it has side effects.
//
// If work is to be preserved across runs, knownStructure should be filled in
// with parenthood information from previously-generated records (for both
// files and directories). Nodes that exist as keys in this map will not be
// re-verified, except to confirm that their content still exists in allScores.
//
// It is expected that the blob store's Load method does score verification for
// us.
func newSuccessorFinder(
	readFiles bool,
	allScores []blob.Score,
	knownStructure map[Node][]Node,
	records chan<- Record,
	clock timeutil.Clock,
	bs blob.Store) (sf graph.SuccessorFinder) {
	typed := &successorFinder{
		readFiles:      readFiles,
		records:        records,
		clock:          clock,
		blobStore:      bs,
		knownScores:    make(map[blob.Score]struct{}),
		knownStructure: knownStructure,
	}

	for _, score := range allScores {
		typed.knownScores[score] = struct{}{}
	}

	sf = typed
	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type successorFinder struct {
	readFiles      bool
	records        chan<- Record
	clock          timeutil.Clock
	blobStore      blob.Store
	knownScores    map[blob.Score]struct{}
	knownStructure map[Node][]Node
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

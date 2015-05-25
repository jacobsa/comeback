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

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/repr"
	"golang.org/x/net/context"
)

// Create a visitor for the DAG of blobs in the supplied bucket. Node names are
// expected to be generated by Node.String.
//
// The visitor reads directory blobs, parses them, and emits their children as
// adjacent nodes. For file nodes, the visitor verifies that their score exists
// (according to allScores), and verifies that the blob can be loaded if
// readFiles is true.
//
// If work is to be preserved across runs, knownStructure should be filled in
// with parenthood information from previously-generated records (for both file
// and directories). Nodes that exist as keys will not be re-verified, except
// to confirm that they still exist in allScores.
//
// A record is written to the supplied channel for every piece of information
// that is verified.
//
// It is expected that the blob store's Load method does score verification for
// us.
func NewVisitor(
	readFiles bool,
	allScores []blob.Score,
	knownStructure map[Node][]Node,
	records chan<- Record,
	clock timeutil.Clock,
	bs blob.Store) (v graph.Visitor) {
	typed := &visitor{
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

	v = typed
	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type visitor struct {
	readFiles      bool
	records        chan<- Record
	clock          timeutil.Clock
	blobStore      blob.Store
	knownScores    map[blob.Score]struct{}
	knownStructure map[Node][]Node
}

func (v *visitor) visitFile(
	ctx context.Context,
	n Node) (err error) {
	// If reading files is disabled, there is nothing further to do.
	if !v.readFiles {
		return
	}

	// Make sure we can load the blob contents. Presumably the blob store
	// verifies the score (of the ciphertext) on the way through.
	_, err = v.blobStore.Load(ctx, n.Score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", n.Score.Hex(), err)
		return
	}

	// Certify that we verified the file piece.
	r := Record{
		Time: v.clock.Now(),
		Node: n,
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case v.records <- r:
	}

	return
}

func (v *visitor) visitDir(
	ctx context.Context,
	n Node) (adjacent []string, err error) {
	// Load the blob contents.
	contents, err := v.blobStore.Load(ctx, n.Score)
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
		Time: v.clock.Now(),
		Node: n,
	}

	for _, entry := range listing {
		var n Node

		// Is this a directory?
		switch entry.Type {
		case fs.TypeFile:
			n.Dir = false

		case fs.TypeDirectory:
			n.Dir = true

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
			n.Score = score
			r.Children = append(r.Children, n)
		}
	}

	// Certify that we verified the directory.
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case v.records <- r:
	}

	// Return child node names.
	for _, child := range r.Children {
		adjacent = append(adjacent, child.String())
	}

	return
}

func (v *visitor) Visit(
	ctx context.Context,
	nodeName string) (adjacent []string, err error) {
	// Parse the node name.
	n, err := ParseNode(nodeName)
	if err != nil {
		err = fmt.Errorf("ParseNode(%q): %v", nodeName, err)
		return
	}

	// Make sure the score actually exists.
	if _, ok := v.knownScores[n.Score]; !ok {
		err = fmt.Errorf("Unknown score for node: %s", n.String())
		return
	}

	// If we have already verified this node, there is nothing further to do.
	if _, ok := v.knownStructure[n]; ok {
		return
	}

	// Perform file or directory-specific logic.
	if n.Dir {
		adjacent, err = v.visitDir(ctx, n)
		return
	} else {
		err = v.visitFile(ctx, n)
		return
	}
}

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
	"github.com/jacobsa/timeutil"
	"golang.org/x/net/context"
)

// Read all blobs necessary for verifying the directory structure rooted at a
// set of backup root scores, ensuring that the entire directory structure is
// intact in GCS.
//
// Optionally, all file content is also read and verified. This is less
// important than verifying directory connectedness if we trust that GCS does
// not corrupt object metadata (where we store expected CRC32C and MD5) and
// does correctly report the object's CRC32C and MD5 sums in listings,
// verifying them periodically.
//
// If work is to be preserved across runs, knownStructure should be filled in
// with parenthood information from previously-generated records (for both
// files and directories). Nodes that exist as keys in this map will not be
// re-verified, except to confirm that their content still exists in allScores.
//
// It is expected that the blob store's Load method does score verification for
// us.
func Verify(
	ctx context.Context,
	readFiles bool,
	rootScores []blob.Score,
	allScores []blob.Score,
	knownStructure map[Node][]Node,
	records chan<- Record,
	blobStore blob.Store) (err error) {
	clock := timeutil.RealClock()

	// Set up a dependency resolver that reads directory listings. It also takes
	// care of confirming that all scores (for files and directories) exist.
	dr := newDependencyResolver(
		allScores,
		knownStructure,
		records,
		blobStore,
		clock)

	// Do we need to do anything for file nodes?
	var visitor dag.Visitor
	if readFiles {
		visitor = newVisitor(records, blobStore, clock)
	} else {
		visitor = &doNothingVisitor{}
	}

	// Traverse the graph.
	var rootNodes []dag.Node
	for _, s := range rootScores {
		n := Node{
			Score: s,
			Dir:   true,
		}

		rootNodes = append(rootNodes, n)
	}

	const parallelism = 128
	err = dag.Visit(ctx, rootNodes, dr, visitor, parallelism)
	if err != nil {
		err = fmt.Errorf("dag.Visit: %v", err)
		return
	}

	return
}

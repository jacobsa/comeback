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
	"github.com/jacobsa/syncutil"
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
	bs blob.Store) (err error) {
	b := syncutil.NewBundle(ctx)

	// Explore the graph starting at the specified roots. Use an "experimentally
	// determined" parallelism, which in theory should depend on bandwidth-delay
	// products but in practice comes down to when the OS gets cranky about open
	// files.
	graphNodes := make(chan graph.Node, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(graphNodes)
		const parallelism = 128

		sf := newSuccessorFinder(
			readFiles,
			allScores,
			knownStructure,
			records,
			timeutil.RealClock(),
			bs)

		var graphRoots []graph.Node
		for _, s := range rootScores {
			n := Node{
				Score: s,
				Dir:   true,
			}

			graphRoots = append(graphRoots, n)
		}

		err = graph.ExploreDirectedGraph(
			ctx,
			sf,
			graphRoots,
			graphNodes,
			parallelism)

		if err != nil {
			err = fmt.Errorf("ExploreDirectedGraph: %v", err)
			return
		}

		return
	})

	// Throw away the graph nodes returned by ExploreDirectedGraph. We don't need
	// them; the successor finder mints records and writes them to the channel
	// for us.
	b.Add(func(ctx context.Context) (err error) {
		for _ = range graphNodes {
		}

		return
	})

	err = b.Join()
	return
}

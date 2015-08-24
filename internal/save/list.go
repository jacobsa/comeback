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
	"fmt"
	"io"
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	"github.com/jacobsa/syncutil"
)

// Given a base directory and a set of exclusions, list the files and
// directories that would be saved by a backup job with the same info in a
// human-readable format. Write the output to the supplied writer.
func List(
	ctx context.Context,
	w io.Writer,
	basePath string,
	exclusions []*regexp.Regexp) (err error) {
	b := syncutil.NewBundle(context.Background())

	// Explore the file system graph, writing all non-excluded nodes into a
	// channel.
	graphNodes := make(chan graph.Node, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(graphNodes)
		sf := newSuccessorFinder(basePath, exclusions)

		const parallelism = 8
		err = graph.ExploreDirectedGraph(
			ctx,
			sf,
			[]graph.Node{(*pathAndFileInfo)(nil)},
			graphNodes,
			parallelism)

		if err != nil {
			err = fmt.Errorf("ExploreDirectedGraph: %v", err)
			return
		}

		return
	})

	// Print out info about each node.
	b.Add(func(ctx context.Context) (err error) {
		for n := range graphNodes {
			pfi, ok := n.(*pathAndFileInfo)
			if !ok {
				err = fmt.Errorf("Unexpected node type: %T", n)
				return
			}

			// Skip the root node.
			if pfi == nil {
				continue
			}

			_, err = fmt.Fprintf(w, "%s %d\n", pfi.Path, pfi.Info.Size())
			if err != nil {
				err = fmt.Errorf("Fprintf: %v", err)
				return
			}
		}

		return
	})

	err = b.Join()
	return
}

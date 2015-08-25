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
	"io"
	"os"
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	"github.com/jacobsa/syncutil"
)

// Given a base directory and a set of exclusions, list all file system nodes
// involved, filling in only the RelPath and Parent fields. Output nodes are
// guaranteed to be in reverse topologically sorted order: children appear
// before their parents.
func listNodes(
	ctx context.Context,
	basePath string,
	exclusions []*regexp.Regexp,
	nodes chan<- *fsNode) (err error) {
	err = errors.New("TODO")
	return
}

// Given a base directory and a set of exclusions, list the files and
// directories that would be saved by a backup job with the same info in a
// human-readable format. Write the output to the supplied writer.
func List(
	ctx context.Context,
	w io.Writer,
	basePath string,
	exclusions []*regexp.Regexp) (err error) {
	// TODO(jacobsa): Make this function use listNodes.

	b := syncutil.NewBundle(context.Background())

	// Explore the file system graph, writing all non-excluded nodes into a
	// channel.
	graphNodes := make(chan graph.Node, 100)
	b.Add(func(ctx context.Context) (err error) {
		// Set up a root node.
		rootNode := &fsNode{
			RelPath: "",
			Parent:  nil,
		}

		rootNode.Info, err = os.Lstat(basePath)
		if err != nil {
			err = fmt.Errorf("os.Lstat: %v", err)
			return
		}

		if !rootNode.Info.IsDir() {
			err = fmt.Errorf("Not a directory: %q", basePath)
			return
		}

		// Explore the graph.
		defer close(graphNodes)
		sf := newSuccessorFinder(basePath, exclusions)

		const parallelism = 8
		err = graph.ExploreDirectedGraph(
			ctx,
			sf,
			[]graph.Node{rootNode},
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
		for graphNode := range graphNodes {
			n, ok := graphNode.(*fsNode)
			if !ok {
				err = fmt.Errorf("Unexpected node type: %T", n)
				return
			}

			// Skip the root node.
			_, err = fmt.Fprintf(w, "%q %d\n", n.RelPath, n.Info.Size())
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

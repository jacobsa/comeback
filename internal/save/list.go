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
	b := syncutil.NewBundle(ctx)

	// Find the nodes in the appropriate order, writing them to a channel of
	// graph.Node.
	graphNodes := make(chan graph.Node, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(graphNodes)

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
		sf := newSuccessorFinder(basePath, exclusions)
		err = graph.ReverseTopsortTree(
			ctx,
			sf,
			rootNode,
			graphNodes)

		if err != nil {
			err = fmt.Errorf("ReverseTopsortTree: %v", err)
			return
		}

		return
	})

	// Convert to *fsNode.
	b.Add(func(ctx context.Context) (err error) {
		for graphNode := range graphNodes {
			n, ok := graphNode.(*fsNode)
			if !ok {
				err = fmt.Errorf("Unexpected node type: %T", graphNode)
				return
			}

			select {
			case nodes <- n:
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		return
	})

	err = b.Join()
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
	b := syncutil.NewBundle(ctx)

	// List nodes.
	nodes := make(chan *fsNode, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(nodes)
		err = listNodes(ctx, basePath, exclusions, nodes)
		if err != nil {
			err = fmt.Errorf("listNodes: %v", err)
			return
		}

		return
	})

	// Print out info about each node.
	b.Add(func(ctx context.Context) (err error) {
		for n := range nodes {
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

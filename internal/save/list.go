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
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/syncutil"
)

// Given a base directory and a set of exclusions, list all file system nodes
// involved.
func listNodes(
	ctx context.Context,
	basePath string,
	exclusions []*regexp.Regexp,
	nodes chan<- *fsNode) (err error) {
	// Visit all nodes in the graph with a visitor that simply writes them to a
	// channel.
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
			_, err = fmt.Fprintf(w, "%q %d\n", n.RelPath, n.Info.Size)
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

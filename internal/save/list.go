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
	"context"
	"fmt"
	"io"
	"regexp"

	"github.com/jacobsa/comeback/internal/dag"
)

// Given a base directory and a set of exclusions, list the files and
// directories that would be saved by a backup job with the same info in a
// human-readable format. Write the output to the supplied writer.
func List(
	ctx context.Context,
	w io.Writer,
	basePath string,
	exclusions []*regexp.Regexp) (err error) {
	// Visit all nodes in the graph with a visitor that prints info about the
	// node.
	dr := newDependencyResolver(basePath, exclusions)
	v := &listVisitor{w: w}

	const resolverParallelism = 1
	const visitorParallelism = 1

	err = dag.Visit(
		ctx,
		[]dag.Node{makeRootNode()},
		dr,
		v,
		resolverParallelism,
		visitorParallelism)

	if err != nil {
		err = fmt.Errorf("dag.Visit: %v", err)
		return
	}

	return
}

type listVisitor struct {
	w io.Writer
}

var _ dag.Visitor = &listVisitor{}

func (v *listVisitor) Visit(ctx context.Context, untyped dag.Node) (err error) {
	// Check the type of the node.
	n, ok := untyped.(*fsNode)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	// Print info about the node.
	_, err = fmt.Fprintf(v.w, "%q %d\n", n.RelPath, n.Info.Size)
	if err != nil {
		err = fmt.Errorf("Fprintf: %v", err)
		return
	}

	return
}

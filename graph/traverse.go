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

package graph

import (
	"errors"

	"golang.org/x/net/context"
)

// A visitor in a directed graph whose nodes are identified by strings.
type Visitor interface {
	// Process the supplied node and return a list of direct successors.
	Visit(ctx context.Context, node string) (adjacent []string, err error)
}

// Invoke v.Visit on each node reachable from the supplied search roots,
// including the roots themselves. Use the supplied degree of parallelism.
//
// It is guaranteed that if a node N is fed to v.Visit, then either:
//
//  *  N is an element of roots, or
//  *  There exists a direct predecessor N' of N such that v.Visit(N') was
//     called and returned successfully.
//
// In particular, if the graph is a rooted tree and searching starts at the
// root, then parents will be successfully visited before children are visited.
// However note that in arbitrary DAGs it is *not* guaranteed that all of a
// node's predecessors have been visited before it is.
func Traverse(
	ctx context.Context,
	parallelism int,
	roots []string,
	v Visitor) (err error) {
	err = errors.New("TODO: Traverse")
	return
}

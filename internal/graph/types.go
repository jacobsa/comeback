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

import "golang.org/x/net/context"

// A graph node consists of arbitrary information useful to the caller. The
// only requirement is that all nodes encountered within an individual call to
// one of the functions in this package be mutually comparable according to the
// Go spec. For example, strings or pointers to a single struct type would be
// good choices.
type Node interface{}

// A SuccessorFinder knows how to find the direct successors of a node within a
// directed graph.
type SuccessorFinder interface {
	// Return a list of all unique direct successors of n within the graph whose
	// structure is defined by this object.
	FindDirectSuccessors(
		ctx context.Context,
		n Node) (successors []Node, err error)
}

// A Visitor knows how to process each node in some graph traversal.
type Visitor interface {
	Visit(ctx context.Context, n Node) (err error)
}

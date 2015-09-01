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

package dag

import "golang.org/x/net/context"

// A value uniquely identifying a node within a DAG. A good choice is an
// integer, a string, or a pointer to a struct. The latter is useful for
// situations where a DependencyResolver or Visitor wants to modify state
// related to a node; the state can be changed without the pointer changing.
//
// All nodes encountered within a call to one of this functions must be
// mutually comparable according to the Go spec.
type Node interface{}

// A DependencyResolver knows how to find the direct dependencies of a node
// within a DAG.
type DependencyResolver interface {
	// Return a list of all unique direct dependencies of n within the DAG whose
	// structure is defined by this object.
	FindDependencies(
		ctx context.Context,
		n Node) (deps []Node, err error)
}

// A Visitor knows how to process each node in a graph traversal.
type Visitor interface {
	Visit(ctx context.Context, n Node) (err error)
}

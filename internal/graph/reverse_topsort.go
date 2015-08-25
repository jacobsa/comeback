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

// Write all of the nodes for the tree rooted at the given node to the supplied
// channel. The order is guaranteed to be a reverse topological sort (i.e. a
// node appears only after all of its successors have appeared).
func ReverseTopsortTree(
	ctx context.Context,
	sf SuccessorFinder,
	root Node,
	nodes chan<- Node) (err error) {
	err = errors.New("TODO")
	return
}

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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/graph"
)

// For each incoming node, use the supplied blob store to ensure that the node
// has a non-nil list of scores. Incoming nodes must be in reverse
// topologically sorted order: children must appear before parents.
func fillInScores(
	ctx context.Context,
	blobStore blob.Store,
	nodeIn <-chan *fsNode,
	nodesOut chan<- *fsNode) (err error) {
	err = errors.New("TODO")
	return
}

// Create a graph.Visitor for *fsNode that saves to the supplied blob store,
// filling in the node's Scores field when it is nil. All visited nodes are
// then written to nodesOut.
func newVisitor(
	blobStore blob.Store,
	nodesOut chan<- *fsNode) (v graph.Visitor) {
	v = &visitor{
		blobStore: blobStore,
		nodesOut:  nodesOut,
	}

	return
}

type visitor struct {
	blobStore blob.Store
	nodesOut  chan<- *fsNode
}

func (v *visitor) Visit(ctx context.Context, untyped graph.Node) (err error) {
	err = errors.New("TODO")
	return
}

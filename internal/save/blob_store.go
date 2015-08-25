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
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/graph"
)

const fileChunkSize = 1 << 24

// For each incoming node, use the supplied blob store to ensure that the node
// has a non-nil list of scores. Incoming nodes must be in reverse
// topologically sorted order: children must appear before parents.
func fillInScores(
	ctx context.Context,
	basePath string,
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
	chunkSize int,
	basePath string,
	blobStore blob.Store,
	nodesOut chan<- *fsNode) (v graph.Visitor) {
	v = &visitor{
		chunkSize: chunkSize,
		basePath:  basePath,
		blobStore: blobStore,
		nodesOut:  nodesOut,
	}

	return
}

type visitor struct {
	chunkSize int
	basePath  string
	blobStore blob.Store
	nodesOut  chan<- *fsNode
}

func (v *visitor) Visit(ctx context.Context, untyped graph.Node) (err error) {
	// Check the type of the node.
	n, ok := untyped.(*fsNode)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	// Ensure that the node has scores set, if it needs to.
	err = v.fillInScores(ctx, n)
	if err != nil {
		err = fmt.Errorf("fillInScores: %v", err)
		return
	}

	// Pass on the node.
	select {
	case v.nodesOut <- n:
	case <-ctx.Done():
		err = ctx.Err()
		return
	}

	return
}

func (v *visitor) fillInScores(
	ctx context.Context,
	n *fsNode) (err error) {
	// If the node already has scores, we're done.
	if n.Scores != nil {
		return
	}

	// Files and directories are the only interesting cases.
	mode := n.Info.Mode()
	switch {
	case mode == 0:
		n.Scores, err = v.saveFile(ctx, n)
		if err != nil {
			err = fmt.Errorf("saveFile: %v", err)
			return
		}

	case mode&os.ModeDir != 0:
		n.Scores, err = v.saveDir(ctx, n)
		if err != nil {
			err = fmt.Errorf("saveDir: %v", err)
			return
		}
	}

	return
}

// Guarantees non-nil result when successful, even for empty list of scores.
func (v *visitor) saveFile(
	ctx context.Context,
	n *fsNode) (scores []blob.Score, err error) {
	scores = make([]blob.Score, 0, 1)

	// Open the file for reading.
	f, err := os.Open(path.Join(v.basePath, n.RelPath))
	if err != nil {
		err = fmt.Errorf("Open: %v", err)
		return
	}

	defer f.Close()

	// Process a chunk at a time.
	buf := make([]byte, v.chunkSize)
	for {
		// Read some data.
		var n int
		n, err = f.Read(buf)

		switch {
		case err == io.EOF:
			// Ignore EOF.
			err = nil

		case err != nil:
			err = fmt.Errorf("Read: %v", err)
			return
		}

		// Are we done?
		if n == 0 {
			break
		}

		// Write out the blob.
		var s blob.Score
		s, err = v.blobStore.Store(ctx, buf[:n])
		if err != nil {
			err = fmt.Errorf("Store: %v", err)
			return
		}

		scores = append(scores, s)
	}

	return
}

func (v *visitor) saveDir(
	ctx context.Context,
	n *fsNode) (scores []blob.Score, err error) {
	err = errors.New("TODO")
	return
}

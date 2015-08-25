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
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/graph"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/syncutil"
)

const fileChunkSize = 1 << 24

// For each incoming node, use the supplied blob store to ensure that the node
// has a non-nil list of scores. Incoming nodes must be in reverse
// topologically sorted order: children must appear before parents.
func fillInScores(
	ctx context.Context,
	basePath string,
	blobStore blob.Store,
	nodesIn <-chan *fsNode,
	nodesOut chan<- *fsNode) (err error) {
	b := syncutil.NewBundle(ctx)

	// Convert *fsNode to graph.Node.
	graphNodes := make(chan graph.Node, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(graphNodes)
		for n := range nodesIn {
			select {
			case graphNodes <- n:
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		return
	})

	// Traverse the inverted graph, requiring children to finish before parents
	// start.
	b.Add(func(ctx context.Context) (err error) {
		sf := &parentSuccessorFinder{}
		v := newVisitor(
			fileChunkSize,
			basePath,
			blobStore,
			nodesOut)

		// Hopefully enough parallelism to keep our CPUs saturated (for encryption,
		// SHA-1 computation, etc.) or our NIC saturated (for GCS traffic),
		// depending on which is the current bottleneck.
		const parallelism = 128

		err = graph.TraverseDAG(
			ctx,
			graphNodes,
			sf,
			v,
			parallelism)

		if err != nil {
			err = fmt.Errorf("TraverseDAG: %v", err)
			return
		}

		return
	})

	err = b.Join()
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
	err = v.setScores(ctx, n)
	if err != nil {
		err = fmt.Errorf("setScores: %v", err)
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

func (v *visitor) setScores(
	ctx context.Context,
	n *fsNode) (err error) {
	// If the node already has scores, we're done.
	if n.Scores != nil {
		return
	}

	// Files and directories are the only interesting cases.
	switch n.Info.Type {
	case fs.TypeFile:
		n.Scores, err = v.saveFile(ctx, path.Join(v.basePath, n.RelPath))
		if err != nil {
			err = fmt.Errorf("saveFile: %v", err)
			return
		}

	case fs.TypeDirectory:
		n.Scores, err = v.saveDir(ctx, n.Children)
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
	path string) (scores []blob.Score, err error) {
	scores = make([]blob.Score, 0, 1)

	// Open the file for reading.
	f, err := os.Open(path)
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
	children []*fsNode) (scores []blob.Score, err error) {
	// Set up a list of directory entries.
	var entries []*fs.DirectoryEntry
	for _, child := range children {
		entries = append(entries, &child.Info)
	}

	// Create a blob describing the directory's contents.
	b, err := repr.MarshalDir(entries)
	if err != nil {
		err = fmt.Errorf("MarshalDir: %v", err)
		return
	}

	// Write out the blob.
	s, err := v.blobStore.Store(ctx, b)
	if err != nil {
		err = fmt.Errorf("Store: %v", err)
		return
	}

	scores = []blob.Score{s}
	return
}

// A successor finder for the inverted file system graph, where children have
// arrows to their parents.
type parentSuccessorFinder struct {
}

func (sf *parentSuccessorFinder) FindDirectSuccessors(
	ctx context.Context,
	untyped graph.Node) (successors []graph.Node, err error) {
	n, ok := untyped.(*fsNode)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	if n.Parent != nil {
		successors = []graph.Node{n.Parent}
	}

	return
}

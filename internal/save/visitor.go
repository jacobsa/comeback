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
	"log"
	"os"
	"path"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/timeutil"
)

const fileChunkSize = 1 << 24

// Create a dag.Visitor for *fsNode that does the following for each node N:
//
//  *  Ensure that nodes are only regular files, directories, and symlinks.
//
//  *  For files, consult the supplied score map to find a list of scores. If
//     the score map doesn't hit, write the file to the blob store to obtain a
//     list of scores, and update the score map.
//
//  *  For directories, write a listing to blob store to obtain a list of
//     scores.
//
//  *  Write all nodes to the supplied channel.
//
func newVisitor(
	chunkSize int,
	basePath string,
	scoreMap state.ScoreMap,
	blobStore blob.Store,
	readFromDiskSem semaphore,
	clock timeutil.Clock,
	logger *log.Logger,
	visitedNodes chan<- *fsNode) (v dag.Visitor) {
	v = &visitor{
		chunkSize:       chunkSize,
		basePath:        basePath,
		scoreMap:        scoreMap,
		blobStore:       blobStore,
		readFromDiskSem: readFromDiskSem,
		clock:           clock,
		logger:          logger,
		visitedNodes:    visitedNodes,
	}

	return
}

type visitor struct {
	chunkSize       int
	basePath        string
	scoreMap        state.ScoreMap
	blobStore       blob.Store
	readFromDiskSem semaphore
	clock           timeutil.Clock
	logger          *log.Logger
	visitedNodes    chan<- *fsNode
}

func (v *visitor) Visit(ctx context.Context, untyped dag.Node) (err error) {
	// Check the type of the node.
	n, ok := untyped.(*fsNode)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	v.logger.Print(n.RelPath)

	// Check that the node is of a supported type.
	switch n.Info.Type {
	case fs.TypeFile:
	case fs.TypeDirectory:
	case fs.TypeSymlink:

	default:
		err = fmt.Errorf("Unsupported node type: %v", n.Info.Type)
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
	case v.visitedNodes <- n:
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
	if n.Info.Scores != nil {
		return
	}

	// Files and directories are the only interesting cases.
	switch n.Info.Type {
	case fs.TypeFile:
		n.Info.Scores, err = v.saveFile(ctx, n)
		if err != nil {
			err = fmt.Errorf("saveFile(%q): %v", n.RelPath, err)
			return
		}

	case fs.TypeDirectory:
		n.Info.Scores, err = v.saveDir(ctx, n.Children)
		if err != nil {
			err = fmt.Errorf("saveDir(%q): : %v", n.RelPath, err)
			return
		}
	}

	return
}

// Guarantees non-nil result when successful, even for empty list of scores.
func (v *visitor) saveFile(
	ctx context.Context,
	n *fsNode) (scores []blob.Score, err error) {
	// Can we short circuit here using the score map?
	scoreMapKey := makeScoreMapKey(n, v.clock)
	if scoreMapKey != nil {
		scores = v.scoreMap.Get(*scoreMapKey)
		if scores != nil {
			return
		}
	}

	// Ensure that our result will be non-nil, even for the empty list.
	scores = make([]blob.Score, 0, 1)

	// Open the file for reading.
	f, err := os.Open(path.Join(v.basePath, n.RelPath))
	if err != nil {
		err = fmt.Errorf("Open: %v", err)
		return
	}

	defer f.Close()

	// Process a chunk at a time.
	for {
		var s blob.Score
		s, err = v.saveFileChunk(ctx, f)

		if err == io.EOF {
			err = nil
			break
		}

		if err != nil {
			return
		}

		scores = append(scores, s)
	}

	// Update the score map if the file is eligible.
	if scoreMapKey != nil {
		v.scoreMap.Set(*scoreMapKey, scores)
	}

	return
}

// Returns io.EOF when the reader is exhausted.
func (v *visitor) saveFileChunk(
	ctx context.Context,
	f *os.File) (s blob.Score, err error) {
	// Wait for permission to allocate memory.
	err = v.readFromDiskSem.Acquire(ctx)
	if err != nil {
		err = fmt.Errorf("acquiring semaphore: %v", err)
		return
	}

	// Release the semaphore when we return if the blob store doesn't release it.
	needToRelease := true
	ctx = markSemAcquired(
		ctx,
		v.readFromDiskSem,
		func() {
			needToRelease = false
		},
	)

	defer func() {
		if needToRelease {
			v.readFromDiskSem.Release()
		}
	}()

	// Read a chunk of data from the file.
	var n int
	buf := make([]byte, v.chunkSize)
	n, err = io.ReadFull(f, buf)

	switch {
	case err == io.EOF:
		// EOF means we're done.
		return

	case err == io.ErrUnexpectedEOF:
		// A short read is fine.
		err = nil

	case err != nil:
		err = fmt.Errorf("Read: %v", err)
		return
	}

	// Encapsulate the data so it can be identified as a file chunk.
	var chunk []byte
	chunk, err = repr.MarshalFile(buf[:n])
	if err != nil {
		err = fmt.Errorf("MarshalFile: %v", err)
		return
	}

	// Write out the blob.
	saveReq := &blob.SaveRequest{
		Blob: chunk,
	}

	s, err = v.blobStore.Save(ctx, saveReq)
	if err != nil {
		err = fmt.Errorf("Store: %v", err)
		return
	}

	return
}

func (v *visitor) saveDir(
	ctx context.Context,
	children []*fsNode) (scores []blob.Score, err error) {
	// Wait for permission to allocate memory.
	err = v.readFromDiskSem.Acquire(ctx)
	if err != nil {
		err = fmt.Errorf("acquiring semaphore: %v", err)
		return
	}

	// Release the semaphore when we return if the blob store doesn't release it.
	needToRelease := true
	ctx = markSemAcquired(
		ctx,
		v.readFromDiskSem,
		func() {
			needToRelease = false
		},
	)

	defer func() {
		if needToRelease {
			v.readFromDiskSem.Release()
		}
	}()

	// Set up a list of directory entries.
	var entries []*fs.FileInfo
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
	storeReq := &blob.SaveRequest{
		Blob: b,
	}

	s, err := v.blobStore.Save(ctx, storeReq)
	if err != nil {
		err = fmt.Errorf("Store: %v", err)
		return
	}

	scores = []blob.Score{s}
	return
}

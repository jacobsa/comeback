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
	"log"
	"os"
	"path"
	"sync"

	"golang.org/x/net/context"

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
	clock timeutil.Clock,
	logger *log.Logger,
	visitedNodes chan<- *fsNode) (v dag.Visitor) {
	v = &visitor{
		chunkSize:    chunkSize,
		basePath:     basePath,
		scoreMap:     scoreMap,
		blobStore:    blobStore,
		clock:        clock,
		logger:       logger,
		visitedNodes: visitedNodes,
	}

	return
}

type visitor struct {
	chunkSize    int
	basePath     string
	scoreMap     state.ScoreMap
	blobStore    blob.Store
	clock        timeutil.Clock
	logger       *log.Logger
	visitedNodes chan<- *fsNode

	mu sync.Mutex

	// A freelist of objects that it may be beneficial to reuse from call to
	// call.
	//
	// GUARDED_BY(mu)
	freelist []*reusableObjects
}

type reusableObjects struct {
	// A buffer into which we read file chunks.
	//
	// INVARIANT: len(buf) is always the visitor's chunk size, and cap(buf) is a
	// bit more, to cover magic bytes added by repr.MarshalFile.
	buf []byte

	// A request for storing a blob. Reusing this allows us to reuse any internal
	// state that the blob store may cache.
	storeReq blob.StoreRequest
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
			err = fmt.Errorf("saveFile: %v", err)
			return
		}

	case fs.TypeDirectory:
		n.Info.Scores, err = v.saveDir(ctx, n.Children)
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
	ro := v.getReusableObjects()
	defer v.putReusableObjects(ro)

	buf := ro.buf
	storeReq := &ro.storeReq

loop:
	for {
		// Read some data.
		var n int
		n, err = io.ReadFull(f, buf)

		switch {
		case err == io.EOF:
			// EOF means we're done.
			err = nil
			break loop

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
		storeReq.Blob = chunk

		var s blob.Score
		s, err = v.blobStore.Store(ctx, storeReq)
		if err != nil {
			err = fmt.Errorf("Store: %v", err)
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

func (v *visitor) saveDir(
	ctx context.Context,
	children []*fsNode) (scores []blob.Score, err error) {
	ro := v.getReusableObjects()
	defer v.putReusableObjects(ro)

	storeReq := &ro.storeReq

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
	storeReq.Blob = b

	s, err := v.blobStore.Store(ctx, storeReq)
	if err != nil {
		err = fmt.Errorf("Store: %v", err)
		return
	}

	scores = []blob.Score{s}
	return
}

// LOCKS_EXCLUDED(v.mu)
func (v *visitor) getReusableObjects() (ro *reusableObjects) {
	// Check the freelist.
	{
		v.mu.Lock()
		l := len(v.freelist)
		if l > 0 {
			ro = v.freelist[l-1]
			v.freelist = v.freelist[:l-1]
		}
		v.mu.Unlock()
	}

	if ro != nil {
		return
	}

	// Otherwise, allocate.
	ro = &reusableObjects{
		buf: make([]byte, v.chunkSize, v.chunkSize+v.chunkSize/2),
	}

	return
}

// LOCKS_EXCLUDED(v.mu)
func (v *visitor) putReusableObjects(ro *reusableObjects) {
	v.mu.Lock()
	v.freelist = append(v.freelist, ro)
	v.mu.Unlock()
}

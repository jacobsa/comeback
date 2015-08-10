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

package comebackfs

import (
	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
)

// Create a read-only file system for browsing the backup rooted by the
// supplied score.
func NewFileSystem(
	rootScore blob.Score,
	blobStore blob.Store) (fs fuseutil.FileSystem, err error) {
	// Create the file system.
	typed := &fileSystem{
		blobStore:   blobStore,
		nextInodeID: fuseops.RootInodeID,
		inodes:      make(map[fuseops.InodeID]inodeRecord),
	}

	fs = typed
	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	typed.Lock()
	defer typed.Unlock()

	// Set up the root inode.
	root := newDirInode(rootAttrs, rootScore, blobStore)
	typed.registerInode(root)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type fileSystem struct {
	fuseutil.NotImplementedFileSystem
	blobStore blob.Store

	/////////////////////////
	// Mutable data
	/////////////////////////

	// LOCK ORDERING:
	//
	// Let FS be the file system. Define a strict partial ordering < by:
	//
	// *   For any inode I,  I < FS.
	// *   For any handle H, H < FS.
	//
	// and follow the rule "acquire B while holding A only if A < B".
	//
	// In other words:
	//
	// *   Don't hold more than one inode or handle lock at a time.
	// *   Don't acquire an inode or handle lock before the file system lock.
	//
	// The intuition is that inode and handle locks are held for long operations,
	// but the file system lock is lightweight and must not be.
	mu syncutil.InvariantMutex

	// The next inode ID we will hand out.
	//
	// INVARIANT: nextInodeID >= fuseops.RootInodeID
	nextInodeID fuseops.InodeID

	// The inodes we currently know.
	//
	// INVARIANT: For all k, k < nextInodeID
	// INVARIANT: For all v, v.lookupCount > 0
	// INVARIANT: For all v, v.inode is *dirInode
	inodes map[fuseops.InodeID]inodeRecord
}

// An inode and its lookup count.
type inodeRecord struct {
	lookupCount uint64
	inode       interface{}
}

// LOCKS_REQUIRED(fs)
func (fs *fileSystem) checkInvariants()

// Register the supplied inode, returning its ID. Set the initial lookup count
// to one.
//
// LOCKS_REQUIRED(fs)
func (fs *fileSystem) registerInode(d *dirInode) (id fuseops.InodeID)

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) Lock() {
	fs.mu.Lock()
}

// LOCKS_REQUIRED(fs)
func (fs *fileSystem) Unlock() {
	fs.mu.Unlock()
}

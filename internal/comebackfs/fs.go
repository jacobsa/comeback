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
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
)

// Create a read-only file system for browsing the backup rooted by the
// supplied score. All inodes will be owned by the supplied UID/GID pair.
func NewFileSystem(
	uid uint32,
	gid uint32,
	rootScore blob.Score,
	blobStore blob.Store) (fs fuseutil.FileSystem, err error) {
	// Create the file system.
	typed := &fileSystem{
		blobStore:   blobStore,
		nextInodeID: fuseops.RootInodeID,
		inodes:      make(map[fuseops.InodeID]*inodeRecord),
		handles:     make(map[fuseops.HandleID]interface{}),
	}

	fs = typed
	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	typed.Lock()
	defer typed.Unlock()

	// Set up the root inode.
	root := newDirInode(
		fuseops.InodeAttributes{
			Nlink: 1,
			Mode:  0500 | os.ModeDir,
			Uid:   uid,
			Gid:   gid,
		},
		rootScore,
		blobStore)

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
	//
	// GUARDED_BY(mu)
	nextInodeID fuseops.InodeID

	// The inodes we currently know.
	//
	// INVARIANT: For all k, k < nextInodeID
	// INVARIANT: For all v, v.lookupCount > 0
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]*inodeRecord

	// The next handle ID we will hand out.
	//
	// GUARDED_BY(mu)
	nextHandleID fuseops.HandleID

	// The handles that are currently in flight.
	//
	// INVARIANT: For all k, k < nextHandleID
	// INVARIANT: For all v, v is *dirHandle
	//
	// GUARDED_BY(mu)
	handles map[fuseops.HandleID]interface{}
}

// An inode and its lookup count.
type inodeRecord struct {
	lookupCount uint64
	in          inode
}

// LOCKS_REQUIRED(fs)
func (fs *fileSystem) checkInvariants() {
	// INVARIANT: nextInodeID >= fuseops.RootInodeID
	if !(fs.nextInodeID >= fuseops.RootInodeID) {
		log.Fatalf("Unexpected nextInodeID: %d", fs.nextInodeID)
	}

	// INVARIANT: For all k, k < nextInodeID
	for k, _ := range fs.inodes {
		if !(k < fs.nextInodeID) {
			log.Fatalf("ID %d not less than nextInodeID %d", k, fs.nextInodeID)
		}
	}

	// INVARIANT: For all v, v.lookupCount > 0
	for k, v := range fs.inodes {
		if !(v.lookupCount > 0) {
			log.Fatalf("Inode %d has invalid lookupCount %d", k, v.lookupCount)
		}
	}

	// INVARIANT: For all k, k < nextHandleID
	for k, _ := range fs.handles {
		if !(k < fs.nextHandleID) {
			log.Fatalf(
				"Handle ID %d not less than nextHandleID %d",
				k,
				fs.nextHandleID)
		}
	}

	// INVARIANT: For all v, v is *dirHandle
	for k, v := range fs.handles {
		if _, ok := v.(*dirHandle); !ok {
			log.Fatalf("Handle ID %d is of type %T", k, v)
		}
	}
}

// Register the supplied inode, returning its ID. Set the initial lookup count
// to one.
//
// LOCKS_REQUIRED(fs)
func (fs *fileSystem) registerInode(in inode) (id fuseops.InodeID) {
	id = fs.nextInodeID
	fs.nextInodeID++

	fs.inodes[id] = &inodeRecord{
		lookupCount: 1,
		in:          in,
	}

	return
}

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

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) GetInodeAttributes(
	ctx context.Context,
	op *fuseops.GetInodeAttributesOp) (err error) {
	// Find the inode.
	fs.Lock()
	rec := fs.inodes[op.Inode]
	fs.Unlock()

	if rec == nil {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	in := rec.in
	in.Lock()
	defer in.Unlock()

	// Get its attributes. We don't care how long the kernel caches them, because
	// we are immutable.
	op.Attributes = in.Attributes()
	op.AttributesExpiration = time.Now().Add(24 * time.Hour)

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ForgetInode(
	ctx context.Context,
	op *fuseops.ForgetInodeOp) (err error) {
	fs.Lock()
	defer fs.Unlock()

	// Find the inode.
	rec := fs.inodes[op.Inode]
	if rec == nil {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	// Decrement its lookup count.
	if rec.lookupCount < op.N {
		log.Fatalf(
			"Inode %d has lookup count %d, decrementing by %d",
			op.Inode,
			rec.lookupCount,
			op.N)
	}

	rec.lookupCount -= op.N
	if rec.lookupCount == 0 {
		delete(fs.inodes, op.Inode)
	}

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) OpenDir(
	ctx context.Context,
	op *fuseops.OpenDirOp) (err error) {
	fs.Lock()
	defer fs.mu.Unlock()

	// Find the inode and extract its score.
	rec, _ := fs.inodes[op.Inode]
	if rec == nil {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	in := rec.in.(*dirInode)
	score := in.Score()

	// Create and stash the handle.
	op.Handle = fs.nextHandleID
	fs.nextHandleID++

	dh := newDirHandle(score, fs.blobStore)
	fs.handles[op.Handle] = dh

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	// Find the handle.
	fs.Lock()
	h, _ := fs.handles[op.Handle]
	fs.mu.Unlock()

	if h == nil {
		log.Fatalf("Handle %d not found", op.Handle)
	}

	dh := h.(*dirHandle)

	// Destroy it.
	dh.Destroy()

	return
}

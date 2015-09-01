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
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	pkgfs "github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/fuse"
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
		uid:         uid,
		gid:         gid,
		blobStore:   blobStore,
		inodes:      make(map[fuseops.InodeID]*inodeRecord),
		fileHandles: make(map[fuseops.HandleID]*fileHandle),
	}

	fs = typed
	typed.mu = syncutil.NewInvariantMutex(typed.checkInvariants)

	// Set up the root inode.
	typed.Lock()
	defer typed.Unlock()

	rootEntry := &pkgfs.FileInfo{
		Type:        pkgfs.TypeDirectory,
		Name:        "",
		Permissions: 0500,
		Inode:       fuseops.RootInodeID,
		Scores:      []blob.Score{rootScore},
	}

	_, err = typed.lookUpOrCreateInode(rootEntry)
	if err != nil {
		err = fmt.Errorf("Creating root inode: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type fileSystem struct {
	fuseutil.NotImplementedFileSystem

	uid uint32
	gid uint32

	blobStore blob.Store

	/////////////////////////
	// Mutable data
	/////////////////////////

	// LOCK ORDERING:
	//
	// Let FS be the file system. Define a strict partial ordering < by:
	//
	// *   For any inode I,  I < FS.
	// *   For any handle H,  H < FS.
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

	// The inodes we currently know, along with the lookup counts. The inode IDs
	// come from the directory listings stored in GCS.
	//
	// INVARIANT: For all v, v.lookupCount > 0
	//
	// GUARDED_BY(mu)
	inodes map[fuseops.InodeID]*inodeRecord

	// The next handle ID that we will assign.
	//
	// GUARDED_BY(mu)
	nextHandleID fuseops.HandleID

	// In-flight file handles.
	//
	// INVARIANT: For each k, k < nextHandleID
	//
	// GUARDED_BY(mu)
	fileHandles map[fuseops.HandleID]*fileHandle
}

// An inode and its lookup count.
type inodeRecord struct {
	lookupCount uint64
	in          inode
}

// LOCKS_REQUIRED(fs)
func (fs *fileSystem) checkInvariants() {
	// INVARIANT: For all v, v.lookupCount > 0
	for k, v := range fs.inodes {
		if !(v.lookupCount > 0) {
			log.Fatalf("Inode %d has invalid lookupCount %d", k, v.lookupCount)
		}
	}

	// INVARIANT: For each k, k < nextHandleID
	for k, _ := range fs.fileHandles {
		if !(k < fs.nextHandleID) {
			log.Fatalf("Unexpected handle ID: %d", k)
		}
	}
}

// Given a directory entry within the file system, look up an inode for the
// entry if it already exists. If not, create and register one. In either case,
// increment the lookup count.
//
// LOCKS_REQUIRED(fs)
func (fs *fileSystem) lookUpOrCreateInode(e *fs.FileInfo) (
	in inode,
	err error) {
	id := fuseops.InodeID(e.Inode)

	// Do we already have an inode with the given ID?
	if rec, ok := fs.inodes[id]; ok {
		in = rec.in
		rec.lookupCount++
		return
	}

	// Create and register one.
	in, err = createInode(e, fs.uid, fs.gid, fs.blobStore)
	if err != nil {
		err = fmt.Errorf("createInode: %v", err)
		return
	}

	fs.inodes[id] = &inodeRecord{
		lookupCount: 1,
		in:          in,
	}

	return
}

// Create an inode for the supplied directory entry. The UID and GID are
// ignored in favor of the the supplied values.
func createInode(
	e *fs.FileInfo,
	uid uint32,
	gid uint32,
	blobStore blob.Store) (in inode, err error) {
	switch e.Type {
	case fs.TypeDirectory:
		// Check the score count.
		if len(e.Scores) != 1 {
			err = fmt.Errorf(
				"Unexpected score count for directory: %d",
				len(e.Scores))
			return
		}

		// Create the inode.
		in = newDirInode(
			fuseops.InodeAttributes{
				Size:  e.Size,
				Nlink: 1,
				Mode:  e.Permissions | os.ModeDir,
				Mtime: e.MTime,
				Ctime: e.MTime,
				Uid:   uid,
				Gid:   gid,
			},
			e.Scores[0],
			blobStore)

		return

	case fs.TypeFile:
		in = newFileInode(
			fuseops.InodeAttributes{
				Size:  e.Size,
				Nlink: 1,
				Mode:  e.Permissions,
				Mtime: e.MTime,
				Ctime: e.MTime,
				Uid:   uid,
				Gid:   gid,
			},
			e.Scores,
			blobStore)

		return

	case fs.TypeSymlink:
		in = newSymlinkInode(
			fuseops.InodeAttributes{
				Size:  e.Size,
				Nlink: 1,
				Mode:  e.Permissions | os.ModeSymlink,
				Mtime: e.MTime,
				Ctime: e.MTime,
				Uid:   uid,
				Gid:   gid,
			},
			e.Target)

		return

	default:
		err = fmt.Errorf("Don't know how to handle type %d", e.Type)
		return
	}
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
func (fs *fileSystem) LookUpInode(
	ctx context.Context,
	op *fuseops.LookUpInodeOp) (err error) {
	// Find the parent.
	fs.Lock()
	parentRec, _ := fs.inodes[op.Parent]
	fs.Unlock()

	if parentRec == nil {
		log.Fatalf("Inode %d not found", op.Parent)
	}

	parent := parentRec.in.(*dirInode)

	// Find an entry for the child within it.
	parent.Lock()
	e, err := parent.LookUpChild(ctx, op.Name)
	parent.Unlock()

	if err != nil {
		err = fmt.Errorf("LookUpChild: %v", err)
		return
	}

	if e == nil {
		err = fuse.ENOENT
		return
	}

	// Find or create the inode.
	fs.Lock()
	in, err := fs.lookUpOrCreateInode(e)
	fs.Unlock()

	if err != nil {
		err = fmt.Errorf("lookUpOrCreateInode: %v", err)
		return
	}

	// Fill out the response.
	in.Lock()
	defer in.Unlock()

	op.Entry.Child = fuseops.InodeID(e.Inode)
	op.Entry.Attributes = in.Attributes()
	op.Entry.AttributesExpiration = time.Now().Add(24 * time.Hour)
	op.Entry.EntryExpiration = time.Now().Add(24 * time.Hour)

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
	// Nothing interesting to do since we don't use directory handles.
	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReadDir(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// Find the inode.
	fs.Lock()
	rec, _ := fs.inodes[op.Inode]
	fs.Unlock()

	if rec == nil {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	d := rec.in.(*dirInode)

	// Read.
	d.Lock()
	err = d.Read(ctx, op)
	d.Unlock()

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReleaseDirHandle(
	ctx context.Context,
	op *fuseops.ReleaseDirHandleOp) (err error) {
	// Nothing to do. We implement this only to suppress "not implemented" errors
	// in the error log.
	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) OpenFile(
	ctx context.Context,
	op *fuseops.OpenFileOp) (err error) {
	fs.Lock()
	defer fs.Unlock()

	// Find the inode.
	rec, _ := fs.inodes[op.Inode]
	if rec == nil {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	f := rec.in.(*fileInode)

	// Create the handle.
	fh := newFileHandle(f.Scores(), fs.blobStore)

	op.Handle = fs.nextHandleID
	fs.nextHandleID++
	fs.fileHandles[op.Handle] = fh

	// Allow the kernel to cache file contents.
	op.KeepPageCache = true

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReadFile(
	ctx context.Context,
	op *fuseops.ReadFileOp) (err error) {
	// Find the handle.
	fs.Lock()
	fh, ok := fs.fileHandles[op.Handle]
	fs.Unlock()

	if !ok {
		log.Fatalf("Handle %d not found", op.Handle)
	}

	// Read from it.
	fh.Lock()
	op.BytesRead, err = fh.ReadAt(ctx, op.Dst, op.Offset)
	fh.Unlock()

	// We're not supposed to return io.EOF.
	if err == io.EOF {
		err = nil
	}

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReleaseFileHandle(
	ctx context.Context,
	op *fuseops.ReleaseFileHandleOp) (err error) {
	fs.Lock()
	defer fs.Unlock()

	fh := fs.fileHandles[op.Handle]
	fh.Destroy()
	delete(fs.fileHandles, op.Handle)

	return
}

// LOCKS_EXCLUDED(fs)
func (fs *fileSystem) ReadSymlink(
	ctx context.Context,
	op *fuseops.ReadSymlinkOp) (err error) {
	// Find the inode.
	fs.Lock()
	rec, ok := fs.inodes[op.Inode]
	fs.Unlock()

	if !ok {
		log.Fatalf("Inode %d not found", op.Inode)
	}

	in := rec.in.(*symlinkInode)

	// Read from it.
	in.Lock()
	op.Target = in.Target()
	in.Unlock()

	return
}

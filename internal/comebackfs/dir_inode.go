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
	"errors"
	"fmt"
	"log"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
)

// Create an inode with the supplied attributes. The supplied score should
// contain the inode's listing.
func newDirInode(
	attrs fuseops.InodeAttributes,
	score blob.Score,
	blobStore blob.Store) (d *dirInode) {
	d = &dirInode{
		blobStore: blobStore,
		score:     score,
		attrs:     attrs,
	}

	d.mu = syncutil.NewInvariantMutex(d.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type dirInode struct {
	blobStore blob.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	score blob.Score
	attrs fuseops.InodeAttributes

	/////////////////////////
	// Mutable data
	/////////////////////////

	mu syncutil.InvariantMutex

	// The children of the directory, or nil if we haven't yet read the blob.
	//
	// INVARIANT: For each k, v, v.Name == k
	// INVARIANT: For each v, v.HardLinkTarget == nil
	//
	// GUARDED_BY(mu)
	children map[string]*fs.FileInfo

	// A listing for the directory, valid when children != nil.
	//
	// GUARDED_BY(mu)
	listing []fuseutil.Dirent
}

func (d *dirInode) checkInvariants() {
	// INVARIANT: For each k, v, v.Name == k
	for k, v := range d.children {
		if v.Name != k {
			log.Fatalf("Name mismatch: %q, %q", v.Name, k)
		}
	}

	// INVARIANT: For each v, v.HardLinkTarget == nil
	for _, v := range d.children {
		if v.HardLinkTarget != nil {
			log.Fatalf("Found a hard link for name %q", v.Name)
		}
	}
}

// Ensure that d.children is non-nil.
//
// LOCKS_REQUIRED(d)
func (d *dirInode) ensureChildren(ctx context.Context) (err error) {
	if d.children != nil {
		return
	}

	// Read the blob.
	contents, err := d.blobStore.Load(ctx, d.score)
	if err != nil {
		err = fmt.Errorf("blobStore.Load: %v", err)
		return
	}

	// Parse it.
	entries, err := repr.UnmarshalDir(contents)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir: %v", err)
		return
	}

	// Index the entries by name.
	children := make(map[string]*fs.FileInfo)
	for _, e := range entries {
		if e.HardLinkTarget != nil {
			err = errors.New("Hard link enountered.")
			return
		}

		if _, ok := children[e.Name]; ok {
			err = fmt.Errorf("Duplicate name: %q", e.Name)
			return
		}

		children[e.Name] = e
	}

	// Also create a listing.
	var listing []fuseutil.Dirent
	for i, e := range entries {
		de := fuseutil.Dirent{
			Offset: fuseops.DirOffset(i + 1),
			Inode:  fuseops.InodeID(e.Inode),
			Name:   e.Name,
			Type:   convertEntryType(e.Type),
		}

		listing = append(listing, de)
	}

	// Update state.
	d.children = children
	d.listing = listing

	return
}

func convertEntryType(t fs.EntryType) fuseutil.DirentType {
	switch t {
	case fs.TypeFile:
		return fuseutil.DT_File

	case fs.TypeDirectory:
		return fuseutil.DT_Directory

	case fs.TypeSymlink:
		return fuseutil.DT_Link

	default:
		return fuseutil.DT_Unknown
	}
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(d)
func (d *dirInode) Lock() {
	d.mu.Lock()
}

// LOCKS_REQUIRED(d)
func (d *dirInode) Unlock() {
	d.mu.Unlock()
}

// Return the score backing the inode's listing's contents. No lock required.
func (d *dirInode) Score() blob.Score {
	return d.score
}

// LOCKS_REQUIRED(d)
func (d *dirInode) Attributes() (attrs fuseops.InodeAttributes) {
	attrs = d.attrs
	return
}

// Serve the supplied read dir op.
//
// LOCKS_REQUIRED(d)
func (d *dirInode) Read(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// Make sure the listing is present.
	err = d.ensureChildren(ctx)
	if err != nil {
		err = fmt.Errorf("ensureChildren: %v", err)
		return
	}

	// Check that the offset is in range.
	if op.Offset > fuseops.DirOffset(len(d.listing)) {
		err = fmt.Errorf("Out of range offset: %d", op.Offset)
		return
	}

	// Write out the entries in range.
	for _, de := range d.listing[op.Offset:] {
		n := fuseutil.WriteDirent(op.Dst[op.BytesRead:], de)
		if n == 0 {
			break
		}

		op.BytesRead += n
	}

	return
}

// Look up the supplied child name, returning an entry if found. If not found,
// return nil.
//
// LOCKS_REQUIRED(d)
func (d *dirInode) LookUpChild(
	ctx context.Context,
	name string) (e *fs.FileInfo, err error) {
	// Make sure the index of children is present.
	err = d.ensureChildren(ctx)
	if err != nil {
		err = fmt.Errorf("ensureChildren: %v", err)
		return
	}

	// Find the appropriate child, if any.
	e = d.children[name]

	return
}

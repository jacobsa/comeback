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
		score:     score,
		blobStore: blobStore,
		attrs:     attrs,
	}

	d.mu = syncutil.NewInvariantMutex(d.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type dirInode struct {
	score     blob.Score
	blobStore blob.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	attrs fuseops.InodeAttributes

	/////////////////////////
	// Mutable data
	/////////////////////////

	mu syncutil.InvariantMutex

	// A list of entries in the directory, or nil if we haven't yet read the
	// blob.
	//
	// INVARIANT: For each v, v.HardLinkTarget == nil
	//
	// GUARDED_BY(mu)
	entries []*fs.DirectoryEntry
}

func (d *dirInode) checkInvariants() {
	// INVARIANT: For each v, v.HardLinkTarget == nil
	for _, v := range d.entries {
		if v.HardLinkTarget != nil {
			log.Fatalf("Found a hard link for name %q", v.Name)
		}
	}
}

// Ensure that d.entries is non-nil.
//
// LOCKS_REQUIRED(d)
func (d *dirInode) ensureEntries(ctx context.Context) (err error) {
	if d.entries != nil {
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

	// We cound on nil being a sentinel, so don't let a nil slice above (for an
	// empty lit of entries) escape.
	if entries == nil {
		entries = make([]*fs.DirectoryEntry, 0, 1)
	}

	// We don't support hard links.
	for _, e := range entries {
		if e.HardLinkTarget != nil {
			err = errors.New("Hard link enountered.")
			return
		}
	}

	d.entries = entries
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
	// Make sure the list of entries is present.
	err = d.ensureEntries(ctx)
	if err != nil {
		err = fmt.Errorf("ensureEntries: %v", err)
		return
	}

	// Check that the offset is in range.
	if op.Offset > fuseops.DirOffset(len(d.entries)) {
		err = fmt.Errorf("Out of range offset: %d", op.Offset)
		return
	}

	// Write out the entries in range.
	for i := op.Offset; i < fuseops.DirOffset(len(d.entries)); i++ {
		e := d.entries[i]
		de := fuseutil.Dirent{
			Offset: i + 1,
			Inode:  fuseops.InodeID(e.Inode),
			Name:   e.Name,
			Type:   convertEntryType(e.Type),
		}

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
	name string) (e *fs.DirectoryEntry, err error) {
	// Make sure the list of entries is present.
	err = d.ensureEntries(ctx)
	if err != nil {
		err = fmt.Errorf("ensureEntries: %v", err)
		return
	}

	// Find the appropriate entry, if any.
	//
	// TODO(jacobsa): Make this efficient.
	for _, candidate := range d.entries {
		if candidate.Name == name {
			e = candidate
			return
		}
	}

	return
}

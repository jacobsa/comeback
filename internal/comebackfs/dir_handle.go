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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/fuse/fuseops"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
)

func newDirHandle(
	score blob.Score,
	blobStore blob.Store) (dh *dirHandle) {
	dh = &dirHandle{
		score:     score,
		blobStore: blobStore,
	}

	dh.mu = syncutil.NewInvariantMutex(dh.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type dirHandle struct {
	score     blob.Score
	blobStore blob.Store

	/////////////////////////
	// Mutable data
	/////////////////////////

	mu syncutil.InvariantMutex

	// A listing of the directory, or nil if we haven't yet read the blob.
	//
	// GUARDED_BY(mu)
	listing []fuseutil.Dirent
}

func (dh *dirHandle) checkInvariants() {
}

// Ensure that dh.listing is non-nil.
//
// LOCKS_REQUIRED(dh)
func (dh *dirHandle) ensureListing(ctx context.Context) (err error) {
	if dh.listing != nil {
		return
	}

	// Read the blob.
	contents, err := dh.blobStore.Load(ctx, dh.score)
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

	// Convert, ensuring that the result is a non-nil slice even if the listing
	// is empty.
	listing := make([]fuseutil.Dirent, 0, len(entries)+1)
	for i, e := range entries {
		// We don't support hard links.
		if e.HardLinkTarget != nil {
			err = errors.New("Hard link enountered.")
			return
		}

		de := fuseutil.Dirent{
			Offset: fuseops.DirOffset(i),
			Name:   e.Name,
			Type:   convertEntryType(e.Type),

			// Return a bogus inode ID for each entry, but not the root inode ID.
			//
			// NOTE(jacobsa): As far as I can tell this is harmless. Minting and
			// returning a real inode ID is difficult because fuse does not count
			// readdir as an operation that increases the inode ID's lookup count and
			// we therefore don't get a forget for it later, but we would like to not
			// have to remember every inode ID that we've ever minted for readdir.
			Inode: fuseops.RootInodeID + 1,
		}

		listing = append(listing, de)
	}

	// Store the result.
	dh.listing = listing

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

// LOCKS_EXCLUDED(dh)
func (dh *dirHandle) Lock() {
	dh.mu.Lock()
}

// LOCKS_REQUIRED(dh)
func (dh *dirHandle) Unlock() {
	dh.mu.Unlock()
}

// Serve the supplied read dir op.
//
// LOCKS_REQUIRED(dh)
func (dh *dirHandle) Read(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	// Make sure the listing is present.
	err = dh.ensureListing(ctx)
	if err != nil {
		err = fmt.Errorf("ensureListing: %v", err)
		return
	}

	// Check that the offset is in range.
	if op.Offset > fuseops.DirOffset(len(dh.listing)) {
		err = fmt.Errorf("Out of range offset: %d", op.Offset)
		return
	}

	// Write out the results.
	for _, de := range dh.listing[op.Offset:] {
		n := fuseutil.WriteDirent(op.Dst, de)
		if n == 0 {
			break
		}

		op.BytesRead += n
	}

	return
}

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
}

func (d *dirInode) checkInvariants() {
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

// LOCKS_REQUIRED(d)
func (d *dirInode) Attributes() (attrs fuseops.InodeAttributes) {
	attrs = d.attrs
	return
}

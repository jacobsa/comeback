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

// Create an inode with the supplied attributes. The supplied scores should
// contain the inode's contents.
func newFileInode(
	attrs fuseops.InodeAttributes,
	scores []blob.Score,
	blobStore blob.Store) (f *fileInode) {
	f = &fileInode{
		scores:    scores,
		blobStore: blobStore,
		attrs:     attrs,
	}

	f.mu = syncutil.NewInvariantMutex(f.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type fileInode struct {
	blobStore blob.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	scores []blob.Score
	attrs  fuseops.InodeAttributes

	/////////////////////////
	// Mutable data
	/////////////////////////

	mu syncutil.InvariantMutex
}

func (f *fileInode) checkInvariants() {
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(f)
func (f *fileInode) Lock() {
	f.mu.Lock()
}

// LOCKS_REQUIRED(f)
func (f *fileInode) Unlock() {
	f.mu.Unlock()
}

// Return the scores backing the inode's contents. No lock required.
func (f *fileInode) Scores() []blob.Score {
	return f.scores
}

// LOCKS_REQUIRED(f)
func (f *fileInode) Attributes() (attrs fuseops.InodeAttributes) {
	attrs = f.attrs
	return
}

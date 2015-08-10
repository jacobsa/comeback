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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/fuse/fuseops"
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
}

func (dh *dirHandle) checkInvariants() {
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

// Throw away any local state. The handle must not be used again.
//
// LOCKS_EXCLUDED(dh)
func (dh *dirHandle) Destroy() {
}

// Serve the supplied read dir op.
//
// LOCKS_REQUIRED(dh)
func (dh *dirHandle) Read(
	ctx context.Context,
	op *fuseops.ReadDirOp) (err error) {
	err = errors.New("TODO")
	return
}

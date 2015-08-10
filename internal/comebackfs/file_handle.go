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
	"os"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/syncutil"
)

func newFileHandle(
	scores []blob.Score,
	blobStore blob.Store) (fh *fileHandle) {
	fh = &fileHandle{
		blobStore: blobStore,
		scores:    scores,
	}

	fh.mu = syncutil.NewInvariantMutex(fh.checkInvariants)

	return
}

////////////////////////////////////////////////////////////////////////
// Internal
////////////////////////////////////////////////////////////////////////

type fileHandle struct {
	blobStore blob.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	scores []blob.Score

	/////////////////////////
	// Mutable data
	/////////////////////////

	mu syncutil.InvariantMutex

	// A file containing the contents of the blobs with the above scores, or nil
	// if they haven't yet been downloaded.
	//
	// GUARDED_BY(mu)
	file *os.File
}

func (fh *fileHandle) checkInvariants() {
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

// LOCKS_EXCLUDED(fh)
func (fh *fileHandle) Lock() {
	fh.mu.Lock()
}

// LOCKS_REQUIRED(fh)
func (fh *fileHandle) Unlock() {
	fh.mu.Unlock()
}

// Destroy any local state. The handle must not be used again.
//
// LOCKS_EXCLUDED(fh)
func (fh *fileHandle) Destroy() {
	// Close the (anonymous) file, which will have the effect of deleting its
	// content.
	if fh.file != nil {
		fh.file.Close()
		fh.file = nil
	}
}

// Like io.ReaderAt, but with context support.
func (fh *fileHandle) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, err error) {
	err = errors.New("TODO")
	return
}

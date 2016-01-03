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
	"os"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/fuse/fsutil"
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

// LOCKS_REQUIRED(fh)
func (fh *fileHandle) checkInvariants() {
}

// LOCKS_REQUIRED(fh)
func (fh *fileHandle) ensureFile(ctx context.Context) (err error) {
	// Is the file already present?
	if fh.file != nil {
		return
	}

	// Create a file.
	f, err := fsutil.AnonymousFile("")
	if err != nil {
		err = fmt.Errorf("AnonymousFile: %v", err)
		return
	}

	defer func() {
		if err != nil {
			f.Close()
		}
	}()

	// Copy in the contents.
	for _, s := range fh.scores {
		var p []byte

		// Load a chunk.
		p, err = fh.blobStore.Load(ctx, s)
		if err != nil {
			err = fmt.Errorf("Load(%s): %v", s.Hex(), err)
			return
		}

		// Unmarshal it.
		p, err = repr.UnmarshalFile(p)
		if err != nil {
			err = fmt.Errorf("UnmarshalFile: %v", err)
			return
		}

		// Write it out.
		_, err = f.Write(p)
		if err != nil {
			err = fmt.Errorf("Write: %v", err)
			return
		}
	}

	// Update state.
	fh.file = f

	return
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
//
// LOCKS_REQUIRED(fh)
func (fh *fileHandle) ReadAt(
	ctx context.Context,
	p []byte,
	offset int64) (n int, err error) {
	// Make sure the local file is present.
	err = fh.ensureFile(ctx)
	if err != nil {
		err = fmt.Errorf("ensureFile: %v", err)
		return
	}

	// Defer to it.
	n, err = fh.file.ReadAt(p, offset)

	return
}

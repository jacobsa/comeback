// Copyright 2012 Aaron Jacobs. All Rights Reserved.
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

package backup

import (
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"io"
	"io/ioutil"
	"runtime"
)

// An object that knows how to save files to some underlying storage.
type FileSaver interface {
	// Save the contents of the supplied reader to underlying storage, returning
	// a list of scores of blobs that should be concatenated in order to recover
	// its contents.
	Save(r io.Reader) (scores []blob.Score, err error)
}

// Create a file saver that uses the supplied blob store, splitting files into
// chunks of the specified size.
func NewFileSaver(store blob.Store, chunkSize uint32) (FileSaver, error) {
	if chunkSize == 0 {
		return nil, fmt.Errorf("Chunk size must be positive.")
	}

	saver := &fileSaver{blobStore: store, chunkSize: chunkSize}
	startWorkers(saver)
	runtime.SetFinalizer(saver, stopWorkers)

	return saver, nil
}

type fileSaver struct {
	blobStore blob.Store
	chunkSize uint32
}

func startWorkers(f *fileSaver) {
}

func stopWorkers(f *fileSaver) {
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader, chunkSize uint32) ([]byte, error) {
	r = io.LimitReader(r, int64(chunkSize))
	return ioutil.ReadAll(r)
}

func (s *fileSaver) Save(r io.Reader) (scores []blob.Score, err error) {
	// Turn the file into chunks, saving each to the blob store.
	scores = []blob.Score{}
	for {
		chunk, err := getChunk(r, s.chunkSize)
		if err != nil {
			return nil, fmt.Errorf("Reading chunk: %v", err)
		}

		// Are we done?
		if len(chunk) == 0 {
			break
		}

		// Store the chunk.
		score, err := s.blobStore.Store(chunk)
		if err != nil {
			return nil, fmt.Errorf("Storing chunk: %v", err)
		}

		scores = append(scores, score)
	}

	return scores, nil
}

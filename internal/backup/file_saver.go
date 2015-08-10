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
	"io"
	"io/ioutil"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
)

// An object that knows how to save files to some underlying storage.
type FileSaver interface {
	// Save the contents of the given path to underlying storage, returning a
	// list of scores of blobs that should be concatenated in order to recover
	// its contents.
	Save(path string) (scores []blob.Score, err error)
}

// Create a file saver that uses the supplied blob store, splitting files into
// chunks of the specified size.
func NewFileSaver(
	store blob.Store,
	chunkSize int,
	fileSystem fs.FileSystem) (s FileSaver, err error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("Chunk size must be positive.")
	}

	s = &fileSaver{
		store,
		chunkSize,
		fileSystem,
	}

	return
}

type fileSaver struct {
	blobStore  blob.Store
	chunkSize  int
	fileSystem fs.FileSystem
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader, chunkSize int) ([]byte, error) {
	r = io.LimitReader(r, int64(chunkSize))
	return ioutil.ReadAll(r)
}

func (s *fileSaver) Save(path string) (scores []blob.Score, err error) {
	var file io.ReadCloser

	// Open the file.
	if file, err = s.fileSystem.OpenForReading(path); err != nil {
		err = fmt.Errorf("OpenForReading: %v", err)
		return
	}

	defer file.Close()

	// Turn the file into chunks, giving them to the blob store one by one.
	for {
		// Read the chunk.
		var chunk []byte
		chunk, err = getChunk(file, s.chunkSize)
		if err != nil {
			err = fmt.Errorf("Reading chunk: %v", err)
			return
		}

		// Are we done?
		if len(chunk) == 0 {
			break
		}

		// Encapsulate the chunk such that it can be identified as a file.
		chunk, err = repr.MarshalFile(chunk)
		if err != nil {
			err = fmt.Errorf("MarshalFile: %v", err)
			return
		}

		// Feed the chunk to the blob store.
		var score blob.Score
		score, err = s.blobStore.Store(context.TODO(), chunk)
		if err != nil {
			err = fmt.Errorf("Store: %v", err)
			return
		}

		scores = append(scores, score)
	}

	return
}

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
)

// An object that knows how to save files to some underlying storage.
type FileSaver interface {
	// Save the contents of the supplied reader to underlying storage, returning
	// a list of scores of blobs that should be concatenated in order to recover
	// its contents.
	Save(r io.Reader) (scores []blob.Score, err error)
}

func NewFileSaver(store blob.Store) (FileSaver, error) {
	return &fileSaver{blobStore: store}, nil
}

type fileSaver struct {
	blobStore blob.Store
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader) ([]byte, error) {
	r = io.LimitReader(r, 1<<24)
	return ioutil.ReadAll(r)
}

func (s *fileSaver) Save(r io.Reader) (scores []blob.Score, err error) {
	// Turn the file into chunks, saving each to the blob store.
	scores = []blob.Score{}
	for {
		chunk, err := getChunk(r)
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

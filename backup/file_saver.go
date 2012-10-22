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
	"github.com/jacobsa/comeback/concurrent"
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

// Create a file saver that uses the supplied blob store, splitting files into
// chunks of the specified size.
func NewFileSaver(store blob.Store, chunkSize uint32) (s FileSaver, err error) {
	if chunkSize == 0 {
		return nil, fmt.Errorf("Chunk size must be positive.")
	}

	const numWorkers = 1
	s = &fileSaver{
		store,
		chunkSize,
		concurrent.NewExecutor(numWorkers),
	}

	return
}

type fileSaver struct {
	blobStore blob.Store
	chunkSize uint32
	executor  concurrent.Executor
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader, chunkSize uint32) ([]byte, error) {
	r = io.LimitReader(r, int64(chunkSize))
	return ioutil.ReadAll(r)
}

func (s *fileSaver) Save(r io.Reader) (scores []blob.Score, err error) {
	type result struct {
		score blob.Score
		err   error
	}

	// Turn the file into chunks, giving them to the blob store in parallel.
	resultChans := make([]<-chan result, 0)

	for {
		// Read the chunk.
		var chunk []byte
		chunk, err = getChunk(r, s.chunkSize)
		if err != nil {
			err = fmt.Errorf("Reading chunk: %v", err)
			return
		}

		// Are we done?
		if len(chunk) == 0 {
			break
		}

		// Add the chunk to the queue of work. Make sure to use a buffered channel
		// for the result so that the work function doesn't block on sending to it
		// while we block on sending to the executor running the work function.
		resultChan := make(chan result, 1)
		resultChans = append(resultChans, resultChan)

		processChunk := func() {
			var r result
			r.score, r.err = s.blobStore.Store(chunk)
			resultChan <- r
		}

		s.executor.Add(processChunk)
	}

	// Read back scores.
	scores = make([]blob.Score, len(resultChans))

	for i, resultChan := range resultChans {
		result := <-resultChan
		if result.err != nil {
			err = fmt.Errorf("Storing chunk %d: %v", i, result.err)
			return
		}

		scores[i] = result.score
	}

	return scores, nil
}

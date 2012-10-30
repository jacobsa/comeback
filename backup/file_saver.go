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
	"github.com/jacobsa/comeback/fs"
	"io"
	"io/ioutil"
)

// An object that knows how to save files to some underlying storage.
type FileSaver interface {
	// Save the contents of the given path to underlying storage, returning a
	// list of scores of blobs that should be concatenated in order to recover
	// its contents.
	SavePath(path string) (scores []blob.Score, err error)
}

// Create a file saver that uses the supplied blob store, splitting files into
// chunks of the specified size. The executor will be used for scheduling calls
// to the blob store, and may be used to control the degree of parallelism in
// those calls.
func NewFileSaver(
	store blob.Store,
	chunkSize uint32,
	fileSystem fs.FileSystem,
	executor concurrent.Executor,
) (s FileSaver, err error) {
	if chunkSize == 0 {
		return nil, fmt.Errorf("Chunk size must be positive.")
	}

	s = &fileSaver{
		store,
		chunkSize,
		fileSystem,
		executor,
	}

	return
}

type fileSaver struct {
	blobStore  blob.Store
	chunkSize  uint32
	fileSystem fs.FileSystem
	executor   concurrent.Executor
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader, chunkSize uint32) ([]byte, error) {
	r = io.LimitReader(r, int64(chunkSize))
	return ioutil.ReadAll(r)
}

func (s *fileSaver) SavePath(path string) (scores []blob.Score, err error) {
	var file io.ReadCloser

	// Open the file.
	if file, err = s.fileSystem.OpenForReading(path); err != nil {
		err = fmt.Errorf("OpenForReading: %v", err)
		return
	}

	defer file.Close()

	// Turn the file into chunks, giving them to the blob store in parallel.
	type result struct {
		score blob.Score
		err   error
	}

	resultChans := make([]<-chan result, 0)

	// Make sure we drain results before returning, even if we return early. This
	// makes it easy to avoid races in tests.
	defer func() {
		for _, c := range resultChans {
			<-c
		}
	}()

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

		// Add the chunk to the queue of work. Make sure to use a buffered channel
		// for the result so that the work function doesn't block on sending to it
		// while we block on sending to the executor running the work function.
		resultChan := make(chan result, 1)
		resultChans = append(resultChans, resultChan)

		processChunk := func() {
			var r result
			r.score, r.err = s.blobStore.Store(chunk)
			resultChan <- r

			// Close the channel since it can no longer be written to.
			close(resultChan)
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

	return
}

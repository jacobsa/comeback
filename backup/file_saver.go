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

type fileSaverResult struct {
	score blob.Score
	err error
}

type fileSaverWork struct {
	data []byte
	resultChan chan<- fileSaverResult
}

type fileSaver struct {
	blobStore blob.Store
	chunkSize uint32
	work chan<- fileSaverWork
}

func startWorkers(s *fileSaver) {
	work := make(chan fileSaverWork)
	s.work = work

	processWork := func() {
		for item := range work {
			score, err := s.blobStore.Store(item.data)
			item.resultChan <- fileSaverResult{score, err}
		}
	}

	const numWorkers = 1
	for i := 0; i < numWorkers; i++ {
		go processWork()
	}
}

func stopWorkers(s *fileSaver) {
	close(s.work)
}

// Read 16 MiB from the supplied reader, returning less iff the reader returns
// an error (including EOF). Do not treat EOF as an error condition.
func getChunk(r io.Reader, chunkSize uint32) ([]byte, error) {
	r = io.LimitReader(r, int64(chunkSize))
	return ioutil.ReadAll(r)
}

func (s *fileSaver) Save(r io.Reader) (scores []blob.Score, err error) {
	// Turn the file into chunks, giving each to the saver's workers to store in
	// the blob store.
	resultChans := make([]<-chan fileSaverResult, 0)
	for {
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
		// for the result so that the workers don't block on sending to it while we
		// block on sending to them.
		resultChan := make(chan fileSaverResult, 1)
		resultChans = append(resultChans, resultChan)

		workItem := fileSaverWork{chunk, resultChan}
		s.work <- workItem
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

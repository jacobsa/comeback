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

package blob

import (
	"fmt"
	"sync"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/gcloud/syncutil"
)

// Return a blob store that stores blobs in the supplied key/value store. Keys
// look like:
//
//     <prefix><score>
//
// where <score> is the result of calling Score.Hex.
//
// The blob store trusts that it has full ownership of this portion of the
// store's key space -- if a score key exists, then it points to the correct
// data.
//
// bytesInFlight and requestsInFlight control the level of parallelism with
// which we will call the KV store.
func NewKvBasedBlobStore(
	kvStore kv.Store,
	prefix string,
	bytesInFlight uint64,
	requestsInFlight int) Store

type kvBasedBlobStore struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	kvStore kv.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	keyPrefix     string
	maxKVRequests int

	/////////////////////////
	// Mutable state
	/////////////////////////

	// If any background write to the KV store has failed, an error that should
	// be returned for all future calls to Store or Flush.
	writeErr     error
	writeErrOnce sync.Once

	// Semaphore for number of bytes in flight.
	bytesInFlight syncutil.WeightedSemaphore

	mu syncutil.InvariantMutex

	// Scores that we currently have in our possession and will eventually write
	// out to the KV store (or writeErr will be set). May include requests that
	// are waiting for admission by requestsInFlight.
	//
	// GUARDED_BY(mu)
	scoresInProgress []blob.Score

	// Semaphore for number of requests actually in flight to the KV store.
	requestsInFlight syncutil.WeightedSemaphore
}

func (s *kvBasedBlobStore) Store(blob []byte) (score Score, err error) {
	score = ComputeScore(blob)

	// Wait for permission to process this blob. Ensure that we hand back the
	// credit below if we don't actually need to store it again.
	s.bytesInFlight.Acquire(uint64(len(blob)))

	needRelease := true
	defer func() {
		if needRelease {
			s.bytesInFlight.Release(uint64(len(blob)))
		}
	}()

	// Do we need to store the blob? If so, register it.
	s.mu.Lock()
	proceed := s.registerIfNecessary(score, blob)
	s.mu.Unlock()

	// Stop if the blob is already in good hands.
	if !proceed {
		return
	}

	needRelease = false

	// Spawn a goroutine that handles the blob.
	go s.handleBlob(score, blob)
	panic("TODO")

	// Choose a key for this blob based on its score.
	key := s.keyPrefix + score.Hex()

	// Don't bother storing the same blob twice.
	alreadyExists, err := s.kvStore.Contains(key)
	if err != nil {
		err = fmt.Errorf("Contains: %v", err)
		return
	}

	if alreadyExists {
		return
	}

	// Store the blob.
	if err = s.kvStore.Set(key, blob); err != nil {
		err = fmt.Errorf("Set: %v", err)
		return
	}

	return
}

func (s *kvBasedBlobStore) Load(score Score) (blob []byte, err error) {
	// Choose the appropriate key.
	key := s.keyPrefix + score.Hex()

	// Call the key/value store.
	if blob, err = s.kvStore.Get(key); err != nil {
		err = fmt.Errorf("Get: %v", err)
		return
	}

	return
}

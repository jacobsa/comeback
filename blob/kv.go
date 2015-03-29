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
// bufferSize controls the number of bytes that may be buffered by this class,
// used to avoid hogging RAM. It should be set to a few times the product of
// the desired bandwidth in bytes and the typical latency of a write to the KV
// store.
//
// maxInFlight controls the maximum parallelism with which we will call the KV
// store, used to avoid hammering it too hard. It should be set to a few times
// the product of the desired request rate in Hz and the typical latency of a
// write.
func NewKvBasedBlobStore(
	kvStore kv.Store,
	prefix string,
	bufferSize int,
	maxInFlight int) Store

type kvBasedBlobStore struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	kvStore kv.Store

	/////////////////////////
	// Constant data
	/////////////////////////

	keyPrefix           string
	maxBytesBuffered    int
	maxRequestsInFlight int

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// If any background write to the KV store has failed, an error that should
	// be returned for all future calls to Store or Flush.
	//
	// GUARDED_BY(mu)
	writeErr error

	// A map from scores that have been accepted by Store but not yet
	// successfully written out to the length of the corresponding blobs.
	//
	// INVARIANT: len(inFlight) <= maxRequestsInFlight
	//
	// GUARDED_BY(mu)
	inFlight map[Score]int

	// A cached sum of the lengths of in-flight scores.
	//
	// INVARIANT: bytesBuffered is the sum of all values of inFlight
	// INVARIANT: bytesBuffered <= maxBytesBuffered
	//
	// GUARDED_BY(mu)
	bytesBuffered int
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

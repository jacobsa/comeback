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

package blob

import (
	"fmt"

	"github.com/jacobsa/gcloud/gcs"
)

// Return a blob store that stores blobs in the supplied GCS bucket. GCS object
// names look like:
//
//     <prefix><score>
//
// where <score> is the result of calling Score.Hex.
//
// The blob store trusts that it has full ownership of this portion of the
// bucket's key space -- if a score name exists, then it points to the correct
// data.
func NewGCSStore(
	bucket gcs.Bucket,
	prefix string) (store Store)

type gcsStore struct {
	bucket    gcs.Bucket
	keyPrefix string
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *kvBasedBlobStore) makeKey(score Score) (key string) {
	key = s.keyPrefix + score.Hex()
	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *kvBasedBlobStore) Store(blob []byte) (score Score, err error) {
	// Will this blob ever fit?
	if len(blob) > s.maxBytesBuffered {
		err = fmt.Errorf(
			"%v-byte blob is larger than buffer size of %v",
			len(blob),
			s.maxBytesBuffered)

		return
	}

	// Compute a score and a key for use under the lock below.
	score = ComputeScore(blob)
	key := s.makeKey(score)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Wait until there is room for this request.
	for !s.hasRoom(len(blob)) {
		s.inFlightChanged.Wait()
	}

	// Have we already decided on a final error?
	if s.writeErr != nil {
		err = fmt.Errorf("Error from previous write: %v", s.writeErr)
		return
	}

	// If we already have a request for this score in flight, we can avoid
	// spawning another one. (If the existing one fails, the user will see an
	// error from Flush.)
	if _, ok := s.inFlight[score]; ok {
		return
	}

	// If the KV store already contains the key, we need not do anything further.
	if s.kvStore.Contains(key) {
		return
	}

	// Mark this blob as in flight.
	s.inFlight[score] = len(blob)
	s.bytesBuffered += len(blob)
	s.inFlightChanged.Broadcast()

	// Spawn a goroutine that will make the blob durable in the background.
	go s.makeDurable(score, key, blob)

	return
}

func (s *kvBasedBlobStore) Flush() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wait until there are no requests in flight.
	for len(s.inFlight) != 0 {
		s.inFlightChanged.Wait()
	}

	err = s.writeErr
	return
}

func (s *kvBasedBlobStore) Contains(score Score) (b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Do we have an in-flight request?
	if _, ok := s.inFlight[score]; ok {
		b = true
		return
	}

	// Does the key exist in the KV store?
	key := s.makeKey(score)
	if s.kvStore.Contains(key) {
		b = true
		return
	}

	return
}

func (s *kvBasedBlobStore) Load(score Score) (blob []byte, err error) {
	// Choose the appropriate key.
	key := s.makeKey(score)

	// Call the key/value store.
	if blob, err = s.kvStore.Get(key); err != nil {
		err = fmt.Errorf("Get: %v", err)
		return
	}

	return
}

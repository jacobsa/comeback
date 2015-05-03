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
	"sync"

	"github.com/jacobsa/gcloud/syncutil"
)

// Return a blob store whose Store method buffers around a wrapped store,
// allowing the caller to proceed concurrently while wrapped.Store runs, even
// if it takes awhile.
//
// bufferSize controls the number of bytes that may be buffered by this store,
// used to avoid hogging RAM. It should be set to a few times the product of
// the desired bandwidth in bytes and the typical latency of a write to the
// wrapped store. This must be at least as large as the largest blob written.
//
// maxInFlight controls the maximum parallelism with which we will call the
// wrapped store, used to avoid hammering it too hard. It should be set to a
// few times the product of the desired request rate in Hz and the typical
// latency of a write.
//
// It is assumed that when wrapped.Store returns successfully, the blob is
// durable.
func NewBufferingStore(
	bufferSize int,
	maxInFlight int,
	wrapped Store) Store {
	s := &bufferingStore{
		wrapped:             wrapped,
		maxBytesBuffered:    bufferSize,
		maxRequestsInFlight: maxInFlight,
		inFlight:            make(map[Score]int),
	}

	s.mu = syncutil.NewInvariantMutex(s.checkInvariants)
	s.inFlightChanged.L = &s.mu

	return s
}

type bufferingStore struct {
	/////////////////////////
	// Dependencies
	/////////////////////////

	wrapped Store

	/////////////////////////
	// Constant data
	/////////////////////////

	maxBytesBuffered    int
	maxRequestsInFlight int

	/////////////////////////
	// Mutable state
	/////////////////////////

	mu syncutil.InvariantMutex

	// If any background write to the wrapped store has failed, an error that
	// should be returned for all future calls to Store or Flush.
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
	// INVARIANT: 0 <= bytesBuffered <= maxBytesBuffered
	//
	// GUARDED_BY(mu)
	bytesBuffered int

	// Signalled when the contents of inFlight change.
	inFlightChanged sync.Cond
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *bufferingStore) checkInvariants() {
	// INVARIANT: len(inFlight) <= maxRequestsInFlight
	if !(len(s.inFlight) <= s.maxRequestsInFlight) {
		panic(fmt.Sprintf("Too many in flight: %v", len(s.inFlight)))
	}

	// INVARIANT: bytesBuffered is the sum of all values of inFlight
	sum := 0
	for _, v := range s.inFlight {
		sum += v
	}

	if sum != s.bytesBuffered {
		panic(fmt.Sprintf("Differing sum: %v vs. %v", sum, s.bytesBuffered))
	}

	// INVARIANT: 0 <= bytesBuffered <= maxBytesBuffered
	if !(0 <= s.bytesBuffered && s.bytesBuffered <= s.maxBytesBuffered) {
		panic(fmt.Sprintf(
			"bytesBuffered of %v not in range [0, %v]",
			s.bytesBuffered,
			s.maxBytesBuffered))
	}
}

// LOCKS_REQURED(s.mu)
func (s *bufferingStore) hasRoom(blobLen int) bool {
	bytesLeft := s.maxBytesBuffered - s.bytesBuffered
	return len(s.inFlight) < s.maxRequestsInFlight && blobLen <= bytesLeft
}

// LOCKS_EXCLUDED(s.mu)
func (s *bufferingStore) makeDurable(score Score, blob []byte) {
	var err error

	// When we exit, set a write error (if appropriate) and update the in-flight
	// map.
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		// Is this the first write error?
		if err != nil && s.writeErr == nil {
			s.writeErr = err
		}

		delete(s.inFlight, score)
		s.bytesBuffered -= len(blob)
		s.inFlightChanged.Broadcast()
	}()

	// Call through to the wrapped store.
	wrappedScore, err := s.wrapped.Store(blob)
	if err != nil {
		return
	}

	// Ensure that the score it returned matches what we computed.
	if wrappedScore != score {
		err = fmt.Errorf("Score mismatch: %v vs. %v", score.Hex(), wrappedScore.Hex())
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *bufferingStore) Store(blob []byte) (score Score, err error) {
	// Will this blob ever fit?
	if len(blob) > s.maxBytesBuffered {
		err = fmt.Errorf(
			"%v-byte blob is larger than buffer size of %v",
			len(blob),
			s.maxBytesBuffered)

		return
	}

	// Compute a score for use under the lock below.
	score = ComputeScore(blob)

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

	// If the wrapped store already contains the score, we need not do anything
	// further.
	if s.wrapped.Contains(score) {
		return
	}

	// Mark this blob as in flight.
	s.inFlight[score] = len(blob)
	s.bytesBuffered += len(blob)
	s.inFlightChanged.Broadcast()

	// Spawn a goroutine that will make the blob durable in the background.
	go s.makeDurable(score, blob)

	return
}

func (s *bufferingStore) Flush() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Wait until there are no requests in flight.
	for len(s.inFlight) != 0 {
		s.inFlightChanged.Wait()
	}

	err = s.writeErr
	return
}

func (s *bufferingStore) Contains(score Score) (b bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Do we have an in-flight request?
	if _, ok := s.inFlight[score]; ok {
		b = true
		return
	}

	// Does the score exist in the wrapped store?
	if s.wrapped.Contains(score) {
		b = true
		return
	}

	return
}

func (s *bufferingStore) Load(score Score) (blob []byte, err error) {
	blob, err = s.wrapped.Load(score)
	return
}

// Copyright 2017 Aaron Jacobs. All Rights Reserved.
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

package save

import (
	"context"

	"github.com/jacobsa/comeback/internal/blob"
)

type semaphore chan struct{}

// Acquire acquires one unit from the semaphore, returning an error and not
// acquiring iff the supplied context is first cancelled. Release must later be
// called when this function returns true.
func (s semaphore) Acquire(ctx context.Context) error {
	select {
	case s <- struct{}{}:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release releases one unit previously acquired.
func (s semaphore) Release() {
	_, ok := <-s
	if !ok {
		panic("releasing a non-acquired semaphore")
	}
}

// semaphoreAcquiringBlobStore wraps another store, acquiring a semaphore
// before calling through to Store.
//
// When the request is handled by a semaphoreReleasingBlobStore, it doesn't do
// anything on the way out. Otherwise it releases the semaphore. It uses
// markSemAcquired to note that the semaphore has been acquired.
type semaphoreAcquiringBlobStore struct {
	blob.Store
	sem semaphore
}

func (bs *semaphoreAcquiringBlobStore) Save(
	ctx context.Context,
	req *blob.SaveRequest) (s blob.Score, err error) {
	// Attempt to acquire the semaphore.
	err = bs.sem.Acquire(ctx)
	if err != nil {
		return
	}

	// Ensure we release the semaphore if a downstream blobstore doesn't first do
	// so.
	needToRelease := true
	ctx = markSemAcquired(
		ctx,
		bs.sem,
		func() {
			needToRelease = false
		},
	)

	defer func() {
		if needToRelease {
			bs.sem.Release()
		}
	}()

	// Call through.
	s, err = bs.Store.Save(ctx, req)
	return
}

// semaphoreReleasingBlobStore wraps another store, using releaseSem to release
// a semapore before calling through to Store.
type semaphoreReleasingBlobStore struct {
	blob.Store
	sem semaphore
}

func (bs *semaphoreReleasingBlobStore) Save(
	ctx context.Context,
	req *blob.SaveRequest) (s blob.Score, err error) {
	// Release.
	releaseSem(ctx, bs.sem)

	// Call through.
	s, err = bs.Store.Save(ctx, req)
	return
}

// markSemAcquired causes releaseSem(ctx, s) to call the supplied function,
// given the context returned. It can be used in a conspiracy to find out
// whether a semaphore needs to be released.
func markSemAcquired(ctx context.Context, s semaphore, f func()) context.Context {
	return context.WithValue(ctx, s, f)
}

// releaseSem(ctx, s) causes s to be released and the function previously
// passed to markSemAcquired along with the context and semaphore to be run.
func releaseSem(ctx context.Context, s semaphore) {
	f := ctx.Value(s)
	if f == nil {
		panic("semaphore not acquired")
	}

	f.(func())()
	s.Release()
}

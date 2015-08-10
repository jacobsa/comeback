// Copyright 2013 Aaron Jacobs. All Rights Reserved.
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

package cache_test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jacobsa/comeback/cache"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

const cacheCapacity = 3

func TestLruCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeKey(s string) (k cache.Key) {
	if len(s) > cache.KeyLength {
		panic(
			fmt.Sprintf(
				"Key length of %d is longer than %d",
				len(s),
				cache.KeyLength))
	}

	copy(k[:], s)
	return
}

type LruCacheTest struct {
	c cache.Cache
}

func init() { RegisterTestSuite(&LruCacheTest{}) }

func (t *LruCacheTest) SetUp(i *TestInfo) {
	t.c = cache.NewLruCache(cacheCapacity)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LruCacheTest) Empty() {
	ExpectEq(nil, t.c.LookUp(makeKey("")))
	ExpectEq(nil, t.c.LookUp(makeKey("taco")))
}

func (t *LruCacheTest) InsertNilValue() {
	ExpectThat(
		func() { t.c.Insert(makeKey("taco"), nil) },
		Panics(HasSubstr("nil value")),
	)
}

func (t *LruCacheTest) LookUpUnknownKey() {
	t.c.Insert(makeKey("burrito"), 17)
	t.c.Insert(makeKey("taco"), 19)

	ExpectEq(nil, t.c.LookUp(makeKey("")))
	ExpectEq(nil, t.c.LookUp(makeKey("enchilada")))
}

func (t *LruCacheTest) FillUpToCapacity() {
	AssertEq(3, cacheCapacity)

	t.c.Insert(makeKey("burrito"), 17)
	t.c.Insert(makeKey("taco"), 19)
	t.c.Insert(makeKey("enchilada"), []byte{0x23, 0x29})

	ExpectEq(17, t.c.LookUp(makeKey("burrito")))
	ExpectEq(19, t.c.LookUp(makeKey("taco")))
	ExpectThat(t.c.LookUp(makeKey("enchilada")), DeepEquals([]byte{0x23, 0x29}))
}

func (t *LruCacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, cacheCapacity)

	t.c.Insert(makeKey("burrito"), 17)
	t.c.Insert(makeKey("taco"), 19)              // Least recent
	t.c.Insert(makeKey("enchilada"), 23)         // Second most recent
	AssertEq(17, t.c.LookUp(makeKey("burrito"))) // Most recent

	// Insert another.
	t.c.Insert(makeKey("queso"), 29)

	// See what's left.
	ExpectEq(nil, t.c.LookUp(makeKey("taco")))
	ExpectEq(17, t.c.LookUp(makeKey("burrito")))
	ExpectEq(23, t.c.LookUp(makeKey("enchilada")))
	ExpectEq(29, t.c.LookUp(makeKey("queso")))
}

func (t *LruCacheTest) Overwrite() {
	t.c.Insert(makeKey("taco"), 17)
	t.c.Insert(makeKey("taco"), 19)
	t.c.Insert(makeKey("taco"), 23)

	ExpectEq(23, t.c.LookUp(makeKey("taco")))

	// The overwritten entries shouldn't count toward capacity.
	AssertEq(3, cacheCapacity)

	t.c.Insert(makeKey("burrito"), 29)
	t.c.Insert(makeKey("enchilada"), 31)

	ExpectEq(23, t.c.LookUp(makeKey("taco")))
	ExpectEq(29, t.c.LookUp(makeKey("burrito")))
	ExpectEq(31, t.c.LookUp(makeKey("enchilada")))
}

func (t *LruCacheTest) SafeForConcurrentAccess() {
	const numWorkers = 8
	runtime.GOMAXPROCS(4)

	// Start a few workers writing to and reading from the cache.
	wg := sync.WaitGroup{}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			const numIters = 1e4
			for i := 0; i < numIters; i++ {
				key := makeKey(fmt.Sprintf("%d", i))
				t.c.Insert(key, i)
				val := t.c.LookUp(key)
				if val != nil && val.(int) != i {
					panic(fmt.Sprintf("Unexpected value: %v", val))
				}
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func (t *LruCacheTest) Encode_EmptyCache() {
	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.c))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded cache.Cache
	AssertEq(nil, decoder.Decode(&decoded))

	ExpectEq(nil, decoded.LookUp(makeKey("")))
	ExpectEq(nil, decoded.LookUp(makeKey("taco")))
}

func (t *LruCacheTest) Encode_PreservesLruOrderAndCapacity() {
	// Contents
	AssertEq(3, cacheCapacity)

	t.c.Insert(makeKey("burrito"), 17)
	t.c.Insert(makeKey("taco"), 19)                      // Least recent
	t.c.Insert(makeKey("enchilada"), []byte{0x23, 0x29}) // Second most recent
	AssertEq(17, t.c.LookUp(makeKey("burrito")))         // Most recent

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.c))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded cache.Cache
	AssertEq(nil, decoder.Decode(&decoded))

	// Insert another.
	decoded.Insert(makeKey("queso"), 29)

	// See what's left.
	ExpectEq(nil, decoded.LookUp(makeKey("taco")))
	ExpectEq(17, decoded.LookUp(makeKey("burrito")))
	ExpectThat(t.c.LookUp(makeKey("enchilada")), DeepEquals([]byte{0x23, 0x29}))
	ExpectEq(29, decoded.LookUp(makeKey("queso")))
}

func (t *LruCacheTest) Encode_ConcurrentAccess() {
	// This test is intended for use with `go test -race`.
	runtime.GOMAXPROCS(4)

	wg := sync.WaitGroup{}
	stopWorkers := make(chan bool)

	// Start a goroutine that continually adds to the cache until told to stop.
	wg.Add(1)
	go func() {
		for i := 0; ; i++ {
			select {
			case <-stopWorkers:
				wg.Done()
				return
			case <-time.After(time.Microsecond):
				t.c.Insert(makeKey(fmt.Sprintf("%d", i)), i)
			}
		}
	}()

	// Make sure the goroutine has a moment to start up.
	<-time.After(1 * time.Millisecond)

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.c))

	// Decode. Nothing bad should happen.
	decoder := gob.NewDecoder(buf)
	var decoded cache.Cache
	AssertEq(nil, decoder.Decode(&decoded))

	close(stopWorkers)
	wg.Wait()
}

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
	"github.com/jacobsa/comeback/cache"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"runtime"
	"sync"
	"testing"
	"time"
)

const cacheCapacity = 3

func TestLruCache(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

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
	ExpectEq(nil, t.c.LookUp(""))
	ExpectEq(nil, t.c.LookUp("taco"))
}

func (t *LruCacheTest) InsertNilValue() {
	ExpectThat(
		func() { t.c.Insert("taco", nil) },
		Panics(HasSubstr("nil value")),
	)
}

func (t *LruCacheTest) LookUpUnknownKey() {
	t.c.Insert("burrito", 17)
	t.c.Insert("taco", 19)

	ExpectEq(nil, t.c.LookUp(""))
	ExpectEq(nil, t.c.LookUp("enchilada"))
}

func (t *LruCacheTest) FillUpToCapacity() {
	AssertEq(3, cacheCapacity)

	t.c.Insert("burrito", 17)
	t.c.Insert("taco", 19)
	t.c.Insert("enchilada", []byte{0x23, 0x29})

	ExpectEq(17, t.c.LookUp("burrito"))
	ExpectEq(19, t.c.LookUp("taco"))
	ExpectThat(t.c.LookUp("enchilada"), DeepEquals([]byte{0x23, 0x29}))
}

func (t *LruCacheTest) ExpiresLeastRecentlyUsed() {
	AssertEq(3, cacheCapacity)

	t.c.Insert("burrito", 17)
	t.c.Insert("taco", 19)  // Least recent
	t.c.Insert("enchilada", 23)  // Second most recent
	AssertEq(17, t.c.LookUp("burrito"))  // Most recent

	// Insert another.
	t.c.Insert("queso", 29)

	// See what's left.
	ExpectEq(nil, t.c.LookUp("taco"))
	ExpectEq(17, t.c.LookUp("burrito"))
	ExpectEq(23, t.c.LookUp("enchilada"))
	ExpectEq(29, t.c.LookUp("queso"))
}

func (t *LruCacheTest) Overwrite() {
	t.c.Insert("taco", 17)
	t.c.Insert("taco", 19)
	t.c.Insert("taco", 23)

	ExpectEq(23, t.c.LookUp("taco"))

	// The overwritten entries shouldn't count toward capacity.
	AssertEq(3, cacheCapacity)

	t.c.Insert("burrito", 29)
	t.c.Insert("enchilada", 31)

	ExpectEq(23, t.c.LookUp("taco"))
	ExpectEq(29, t.c.LookUp("burrito"))
	ExpectEq(31, t.c.LookUp("enchilada"))
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
				key := fmt.Sprintf("%d", i)
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

	ExpectEq(nil, decoded.LookUp(""))
	ExpectEq(nil, decoded.LookUp("taco"))
}

func (t *LruCacheTest) Encode_PreservesLruOrderAndCapacity() {
	// Contents
	AssertEq(3, cacheCapacity)

	t.c.Insert("burrito", 17)
	t.c.Insert("taco", 19)  // Least recent
	t.c.Insert("enchilada", 23)  // Second most recent
	AssertEq(17, t.c.LookUp("burrito"))  // Most recent

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.c))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded cache.Cache
	AssertEq(nil, decoder.Decode(&decoded))

	// Insert another.
	decoded.Insert("queso", 29)

	// See what's left.
	ExpectEq(nil, decoded.LookUp("taco"))
	ExpectEq(17, decoded.LookUp("burrito"))
	ExpectEq(23, decoded.LookUp("enchilada"))
	ExpectEq(29, decoded.LookUp("queso"))
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
				t.c.Insert(fmt.Sprintf("%d", i), i)
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

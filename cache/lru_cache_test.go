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
	"github.com/jacobsa/comeback/cache"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
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
	AssertEq("TODO", "")
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
	AssertEq("TODO", "")
}

func (t *LruCacheTest) Overwrite() {
	AssertEq("TODO", "")
}

func (t *LruCacheTest) SafeForConcurrentAccess() {
	AssertEq("TODO", "")
}

func (t *LruCacheTest) EncodeAndDecode() {
	AssertEq("TODO", "")
}

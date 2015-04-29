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

package cache

import (
	"encoding/gob"

	"github.com/jacobsa/gcloud/syncutil"
	"github.com/jacobsa/util/lrucache"
)

// Create a cache that holds the given number of items, evicting the least
// recently used item when more space is needed.
func NewLruCache(capacity uint) Cache {
	c := &lruCache{
		wrapped: lrucache.New(int(capacity)),
	}

	c.mu = syncutil.NewInvariantMutex(c.wrapped.CheckInvariants)
	return c
}

type lruCache struct {
	mu syncutil.InvariantMutex

	// GUARDED_BY(mu)
	wrapped lrucache.Cache
}

func (c *lruCache) Insert(key Key, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.wrapped.Insert(string(key[:]), value)
}

func (c *lruCache) LookUp(key Key) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.wrapped.LookUp(string(key[:]))
}

////////////////////////////////////////////////////////////////////////
// Gob encoding
////////////////////////////////////////////////////////////////////////

func init() {
	// Make sure that lruCaches can be encoded where Cache interface variables
	// are expected.
	gob.Register(&lruCache{})
}

func (c *lruCache) GobEncode() (b []byte, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simply encode the wrapped cache. Our tricky part is on the decode path.
	b, err = c.wrapped.GobEncode()
	return
}

func (c *lruCache) GobDecode(b []byte) (err error) {
	// Decode the wrapped cache.
	err = c.wrapped.GobDecode(b)
	if err != nil {
		return
	}

	// Initialize the mutex.
	c.mu = syncutil.NewInvariantMutex(c.wrapped.CheckInvariants)

	return
}

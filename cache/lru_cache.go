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
	"container/list"
	"sync"
)

// Create a cache that holds the given number of items, evicting the least
// recently used item when more space is needed.
func NewLruCache(capacity uint) Cache {
	return nil
}

type lruCache struct {
	mutex    sync.RWMutex
	capacity uint

	// List of elements, with least recently used at the tail.
	elems list.List

	// Index int `elems` for lookup by key.
	index map[string]*list.Element
}

func (c *lruCache) Insert(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// If the key is already present, erase it first.
	if _, ok := c.index[key]; ok {
		c.erase_Locked(key)
	}

	// Add a list element and index it.
	elem := c.elems.PushFront(value)
	index[key] = elem
}

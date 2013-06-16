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
	"bytes"
	"container/list"
	"encoding/gob"
	"fmt"
	"sync"
)

// Create a cache that holds the given number of items, evicting the least
// recently used item when more space is needed. The capacity
func NewLruCache(capacity uint) Cache {
	result := &lruCache{}
	result.init(capacity)
	return result
}

type lruCacheElement struct {
	Key string
	Value interface{}
}

type lruCache struct {
	mutex    sync.RWMutex
	capacity uint

	// List of elements, with least recently used at the tail. The type of list
	// values is lruCacheElement.
	elems list.List

	// Index into `elems` for lookup by key.
	index map[string]*list.Element
}

func (c *lruCache) init(capacity uint) {
	if c.index != nil {
		panic("lruCache.init called twice?")
	}

	if capacity == 0 {
		panic("Capacity must be non-zero.")
	}

	c.capacity = capacity
	c.index = make(map[string]*list.Element)
}

func (c *lruCache) Insert(key string, value interface{}) {
	c.mutex.Lock()
	defer c.checkInvariantsAndUnlock()

	// If we allowed inserting nil values, LookUp's semantics wouldn't make sense.
	if value == nil {
		panic("Cannot insert nil value.")
	}

	// Make sure the key isn't already present.
	c.erase_Locked(key)

	// Add a list element and index it.
	elem := c.elems.PushFront(&lruCacheElement{key, value})
	c.index[key] = elem

	// Expire the least recently used element if necessary.
	if uint(len(c.index)) > c.capacity {
		c.erase_Locked(c.elems.Back().Value.(*lruCacheElement).Key)
	}
}

func (c *lruCache) erase_Locked(key string) {
	elem, ok := c.index[key]
	if !ok {
		return
	}

	delete(c.index, key)
	c.elems.Remove(elem)
}

func (c *lruCache) checkInvariantsAndUnlock() {
	// Don't mask other panics.
	if r := recover(); r != nil {
		panic(r)
	}

	if uint(len(c.index)) > c.capacity {
		panic(
			fmt.Sprintf(
				"Index length greater than capacity: %d vs. %d",
				len(c.index),
				c.capacity))
	}

	if len(c.index) != c.elems.Len() {
		panic(
			fmt.Sprintf(
				"Index length doesn't match list length: %d vs. %d",
				len(c.index),
				c.elems.Len()))
	}

	c.mutex.Unlock()
}

func (c *lruCache) LookUp(key string) interface{} {
	c.mutex.Lock()
	defer c.checkInvariantsAndUnlock()

	if elem, ok := c.index[key]; ok {
		c.elems.MoveToFront(elem)
		return elem.Value.(*lruCacheElement).Value
	}

	return nil
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
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	// Encode the capacity.
	if err = encoder.Encode(c.capacity); err != nil {
		err = fmt.Errorf("Encoding capacity: %v", err)
		return
	}

	// Encode the cached elements.
	elemsSlice := make([]*lruCacheElement, 0, c.elems.Len())
	for e := c.elems.Front(); e != nil; e = e.Next() {
		elemsSlice = append(elemsSlice, e.Value.(*lruCacheElement))
	}

	if err = encoder.Encode(elemsSlice); err != nil {
		err = fmt.Errorf("Encoding elems: %v", err)
		return
	}

	b = buf.Bytes()
	return
}

func (c *lruCache) GobDecode(b []byte) (err error) {
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	// Decode the capacity.
	var capacity uint
	if err = decoder.Decode(&capacity); err != nil {
		err = fmt.Errorf("Decoding capacity: %v", err)
		return
	}

	// Initialize the receiver.
	c.init(capacity)

	// Decode the elements.
	var elemsSlice []*lruCacheElement
	if err = decoder.Decode(&elemsSlice); err != nil {
		err = fmt.Errorf("Decoding elems: %v", err)
		return
	}

	return
}

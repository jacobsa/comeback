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

const KeyLength = 16

type Key [KeyLength]byte

// A cache mapping from string keys to arbitrary values. Safe for concurrent
// access, and supports gob encoding.
type Cache interface {
	// Insert the given value into the cache. The value must not be the nil
	// interface, and its type must have been previously registered using
	// gob.Register.
	Insert(key Key, value interface{})

	// Look up a previously-inserted value fo the given key. Return nil if no
	// value is present.
	LookUp(key Key) interface{}
}

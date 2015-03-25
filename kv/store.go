// Copyright 2012 Aaron Jacobs. All Rights Reserved.
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

package kv

// A Store knows how to store keys keyed on values. The format of keys and
// values is left unspecified.
type Store interface {
	// Set the supplied value for the given key, overwriting any existing value.
	Set(key string, val []byte) error

	// Get the most recent value for the supplied key, returning an error if the
	// key is unknown.
	Get(key string) (val []byte, err error)

	// Return true only if the store contains a value for the supplied key. If it
	// is unknown whether the key exists, the store is permitted to return false.
	Contains(key string) (res bool, err error)

	// List all keys with the given prefix.
	ListKeys(prefix string) (keys []string, err error)
}

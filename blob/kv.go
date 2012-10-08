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

package blob

import (
	"github.com/jacobsa/comeback/kv"
)

// Return a blob store that stores blobs in the supplied key/value store. It
// trusts that it has full ownership of the store's key space -- if a score key
// exists, then it points to the correct data.
func NewKvBasedBlobStore(kvStore kv.Store) Store {
	return &kvBasedBlobStore{kvStore}
}

type kvBasedBlobStore struct {
	kvStore kv.Store
}

func (s *kvBasedBlobStore) Store(blob []byte) (score Score, err error)
func (s *kvBasedBlobStore) Load(score Score) (blob []byte, err error)

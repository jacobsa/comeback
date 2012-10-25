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

import (
	"fmt"
	"github.com/jacobsa/comeback/state"
)

// Return a key/value store that reads keys from the supplied string set to
// serve Contains requests, and updates the sets upon successful calls to the
// wrapped store's Set method.
func NewExistingKeysStore(
	existingKeys state.StringSet,
	wrapped Store,
) (store Store) {
	return &existingKeysStore{existingKeys, wrapped}
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type existingKeysStore struct {
	keys state.StringSet
	wrapped Store
}

func (s *existingKeysStore) Set(key []byte, val []byte) (err error) {
	err = fmt.Errorf("TODO")
	return
}

func (s *existingKeysStore) Get(key []byte) (val []byte, err error) {
	err = fmt.Errorf("TODO")
	return
}

func (s *existingKeysStore) Contains(key []byte) (res bool, err error) {
	err = fmt.Errorf("TODO")
	return
}

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

package kv_test

import (
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/kv/mock"
	"github.com/jacobsa/comeback/state"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestExisting(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ExistingKeysStoreTest struct {
	existingKeys state.StringSet
	wrapped mock_kv.MockStore
	store kv.Store
}

func init() { RegisterTestSuite(&ExistingKeysStoreTest{}) }

func (t *ExistingKeysStoreTest) SetUp(i *TestInfo) {
	t.existingKeys = state.NewStringSet()
	t.wrapped = mock_kv.NewMockStore(i.MockController, "wrapped")
	t.store = kv.NewExistingKeysStore(t.existingKeys, t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// Set
////////////////////////////////////////////////////////////////////////

type ExistingKeysStore_SetTest struct {
	ExistingKeysStoreTest

	key []byte
	val []byte
	err error
}

func (t *ExistingKeysStore_SetTest) call() {
	t.err = t.store.Set(t.key, t.val)
}

func (t *ExistingKeysStore_SetTest) CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *ExistingKeysStore_SetTest) WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *ExistingKeysStore_SetTest) WrappedSucceeds() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Contains
////////////////////////////////////////////////////////////////////////

type ExistingKeysStore_ContainsTest struct {
	ExistingKeysStoreTest

	key []byte
	res bool
	err error
}

func (t *ExistingKeysStore_ContainsTest) call() {
	t.res, t.err = t.store.Contains(t.key)
}

func (t *ExistingKeysStore_ContainsTest) KeyInSet() {
	ExpectEq("TODO", "")
}

func (t *ExistingKeysStore_ContainsTest) KeyNotInSet() {
	ExpectEq("TODO", "")
}

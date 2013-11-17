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

package blob_test

import (
	"errors"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/kv/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestKv(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type kvBasedStoreTest struct {
	kvBasedStore mock_kv.MockStore
	store        blob.Store
}

func (t *kvBasedStoreTest) SetUp(i *TestInfo) {
	t.kvBasedStore = mock_kv.NewMockStore(i.MockController, "kvBasedStore")
	t.store = blob.NewKvBasedBlobStore(t.kvBasedStore)
}

////////////////////////////////////////////////////////////////////////
// Store
////////////////////////////////////////////////////////////////////////

type KvBasedStore_StoreTest struct {
	kvBasedStoreTest

	data  []byte
	score blob.Score
	err   error
}

func init() { RegisterTestSuite(&KvBasedStore_StoreTest{}) }

func (t *KvBasedStore_StoreTest) callStore() {
	t.score, t.err = t.store.Store(t.data)
}

func (t *KvBasedStore_StoreTest) CallsContains() {
	t.data = []byte("hello")
	expectedKey := []byte("blob:aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d")

	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(DeepEquals(expectedKey)).
		WillOnce(oglemock.Return(false, errors.New("")))

	// Call
	t.callStore()
}

func (t *KvBasedStore_StoreTest) ContainsReturnsError() {
	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(Any()).
		WillOnce(oglemock.Return(false, errors.New("taco")))

	// Call
	t.callStore()

	ExpectThat(t.err, Error(HasSubstr("Contains")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *KvBasedStore_StoreTest) ContainsSaysYes() {
	t.data = []byte("hello")

	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(Any()).
		WillOnce(oglemock.Return(true, nil))

	// Call
	t.callStore()

	AssertEq(nil, t.err)
	ExpectThat(t.score, DeepEquals(blob.ComputeScore(t.data)))
}

func (t *KvBasedStore_StoreTest) CallsSet() {
	t.data = []byte("hello")
	expectedKey := []byte("blob:aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d")

	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(Any()).
		WillOnce(oglemock.Return(false, nil))

	// Set
	ExpectCall(t.kvBasedStore, "Set")(DeepEquals(expectedKey), DeepEquals(t.data)).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callStore()
}

func (t *KvBasedStore_StoreTest) SetReturnsError() {
	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(Any()).
		WillOnce(oglemock.Return(false, nil))

	// Set
	ExpectCall(t.kvBasedStore, "Set")(Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	t.callStore()

	ExpectThat(t.err, Error(HasSubstr("Set")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *KvBasedStore_StoreTest) SetSaysOkay() {
	t.data = []byte("hello")

	// Contains
	ExpectCall(t.kvBasedStore, "Contains")(Any()).
		WillOnce(oglemock.Return(false, nil))

	// Set
	ExpectCall(t.kvBasedStore, "Set")(Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	t.callStore()

	AssertEq(nil, t.err)
	ExpectThat(t.score, DeepEquals(blob.ComputeScore(t.data)))
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type KvBasedStore_LoadTest struct {
	kvBasedStoreTest

	score blob.Score
	data  []byte
	err   error
}

func init() { RegisterTestSuite(&KvBasedStore_LoadTest{}) }

func (t *KvBasedStore_LoadTest) callStore() {
	t.data, t.err = t.store.Load(t.score)
}

func (t *KvBasedStore_LoadTest) CallsGet() {
	t.score = blob.ComputeScore([]byte("hello"))
	expectedKey := []byte("blob:aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d")

	// Get
	ExpectCall(t.kvBasedStore, "Get")(DeepEquals(expectedKey)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callStore()
}

func (t *KvBasedStore_LoadTest) GetReturnsError() {
	// Get
	ExpectCall(t.kvBasedStore, "Get")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callStore()

	ExpectThat(t.err, Error(HasSubstr("Get")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *KvBasedStore_LoadTest) GetSucceeds() {
	// Get
	returnedData := []byte{0xde, 0xad}
	ExpectCall(t.kvBasedStore, "Get")(Any()).
		WillOnce(oglemock.Return(returnedData, nil))

	// Call
	t.callStore()

	AssertEq(nil, t.err)
	ExpectThat(t.data, DeepEquals(returnedData))
}

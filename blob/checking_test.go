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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestChecking(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type checkingStoreTest struct {
	wrapped mock_blob.MockStore
	store   blob.Store
}

func (t *checkingStoreTest) SetUp(i *TestInfo) {
	t.wrapped = mock_blob.NewMockStore(i.MockController, "wrapped")
	t.store = blob.NewCheckingStore(t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// Store
////////////////////////////////////////////////////////////////////////

type CheckingStore_StoreTest struct {
	checkingStoreTest
}

func init() { RegisterTestSuite(&CheckingStore_StoreTest{}) }

func (t *CheckingStore_StoreTest) CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_StoreTest) WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_StoreTest) WrappedReturnsIncorrectScore() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_StoreTest) WrappedReturnsCorrectScore() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type CheckingStore_LoadTest struct {
	checkingStoreTest
}

func init() { RegisterTestSuite(&CheckingStore_LoadTest{}) }

func (t *CheckingStore_LoadTest) CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_LoadTest) WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_LoadTest) WrappedReturnsCorrectData() {
	ExpectEq("TODO", "")
}

func (t *CheckingStore_LoadTest) WrappedReturnsIncorrectData() {
	ExpectEq("TODO", "")
}

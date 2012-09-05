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
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestEncrypting(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type encryptingStoreTest struct{
	crypter mock_crypto.MockCrypter
	wrapped mock_blob.MockStore
	store blob.Store
}

func (t *encryptingStoreTest) SetUp(i *TestInfo) {
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.wrapped = mock_blob.NewMockStore(i.MockController, "wrapped")
	t.store = blob.NewEncryptingStore(t.crypter, t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// Store
////////////////////////////////////////////////////////////////////////

type StoreTest struct {
	encryptingStoreTest
}

func init() { RegisterTestSuite(&StoreTest{}) }

func (t *StoreTest) CallsCrypter() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) CrypterReturnsError() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) WrappedSucceeds() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type LoadTest struct {
	encryptingStoreTest
}

func init() { RegisterTestSuite(&LoadTest{}) }

func (t *LoadTest) CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) CallsCrypter() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) CrypterReturnsError() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) CrypterSucceeds() {
	ExpectEq("TODO", "")
}

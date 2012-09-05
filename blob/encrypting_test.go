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
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
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
	blob := []byte{0xde, 0xad}

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(DeepEquals(blob)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Store(blob)
}

func (t *StoreTest) CrypterReturnsError() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Store([]byte{})

	ExpectThat(err, Error(HasSubstr("Encrypt")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *StoreTest) CallsWrapped() {
	// Crypter
	encryptedBlob := []byte{0xde, 0xad}

	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(encryptedBlob, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Store")(DeepEquals(encryptedBlob)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Store([]byte{})
}

func (t *StoreTest) WrappedReturnsError() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Store([]byte{})

	ExpectThat(err, Error(Equals("taco")))
}

func (t *StoreTest) WrappedSucceeds() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Wrapped
	expected := blob.ComputeScore([]byte("taco"))

	ExpectCall(t.wrapped, "Store")(Any()).
		WillOnce(oglemock.Return(expected, nil))

	// Call
	score, err := t.store.Store([]byte{})
	AssertEq(nil, err)

	ExpectEq(expected, score)
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type LoadTest struct {
	encryptingStoreTest
}

func init() { RegisterTestSuite(&LoadTest{}) }

func (t *LoadTest) CallsWrapped() {
	score := blob.ComputeScore([]byte("taco"))

	// Wrapped
	ExpectCall(t.wrapped, "Load")(score).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(score)
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

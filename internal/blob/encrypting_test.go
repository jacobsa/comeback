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
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	"github.com/jacobsa/comeback/internal/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestEncrypting(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type encryptingStoreTest struct {
	ctx     context.Context
	crypter mock_crypto.MockCrypter
	wrapped mock_blob.MockStore
	store   blob.Store
}

func (t *encryptingStoreTest) SetUp(i *TestInfo) {
	t.ctx = i.Ctx
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.wrapped = mock_blob.NewMockStore(i.MockController, "wrapped")
	t.store = blob.NewEncryptingStore(t.crypter, t.wrapped)
}

////////////////////////////////////////////////////////////////////////
// Store
////////////////////////////////////////////////////////////////////////

type EncryptingStore_StoreTest struct {
	encryptingStoreTest
}

func init() { RegisterTestSuite(&EncryptingStore_StoreTest{}) }

func (t *EncryptingStore_StoreTest) CallsCrypter() {
	blob := []byte{0xde, 0xad}

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(DeepEquals(blob)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Store(t.ctx, blob)
}

func (t *EncryptingStore_StoreTest) CrypterReturnsError() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Store(t.ctx, []byte{})

	ExpectThat(err, Error(HasSubstr("Encrypt")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *EncryptingStore_StoreTest) CallsWrapped() {
	// Crypter
	encryptedBlob := []byte{0xde, 0xad}

	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(encryptedBlob, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any(), DeepEquals(encryptedBlob)).
		WillOnce(oglemock.Return(blob.Score{}, errors.New("")))

	// Call
	t.store.Store(t.ctx, []byte{})
}

func (t *EncryptingStore_StoreTest) WrappedReturnsError() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any(), Any()).
		WillOnce(oglemock.Return(blob.Score{}, errors.New("taco")))

	// Call
	_, err := t.store.Store(t.ctx, []byte{})

	ExpectThat(err, Error(Equals("taco")))
}

func (t *EncryptingStore_StoreTest) WrappedSucceeds() {
	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Wrapped
	expected := blob.ComputeScore([]byte("taco"))

	ExpectCall(t.wrapped, "Store")(Any(), Any()).
		WillOnce(oglemock.Return(expected, nil))

	// Call
	score, err := t.store.Store(t.ctx, []byte{})
	AssertEq(nil, err)

	ExpectThat(score, DeepEquals(expected))
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type EncryptingStore_LoadTest struct {
	encryptingStoreTest
}

func init() { RegisterTestSuite(&EncryptingStore_LoadTest{}) }

func (t *EncryptingStore_LoadTest) CallsWrapped() {
	score := blob.ComputeScore([]byte("taco"))

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), DeepEquals(score)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(t.ctx, score)
}

func (t *EncryptingStore_LoadTest) WrappedReturnsError() {
	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Load(t.ctx, blob.ComputeScore([]byte{}))

	ExpectThat(err, Error(Equals("taco")))
}

func (t *EncryptingStore_LoadTest) CallsCrypter() {
	// Wrapped
	ciphertext := []byte{0xde, 0xad}

	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return(ciphertext, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(DeepEquals(ciphertext)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(t.ctx, blob.ComputeScore([]byte{}))
}

func (t *EncryptingStore_LoadTest) CrypterReturnsError() {
	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Load(t.ctx, blob.ComputeScore([]byte{}))

	ExpectThat(err, Error(HasSubstr("Decrypt")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *EncryptingStore_LoadTest) CrypterSucceeds() {
	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Crypter
	expected := []byte{0xde, 0xad}

	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return(expected, nil))

	// Call
	blob, err := t.store.Load(t.ctx, blob.ComputeScore([]byte{}))
	AssertEq(nil, err)

	ExpectThat(blob, DeepEquals(expected))
}

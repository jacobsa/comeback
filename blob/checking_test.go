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
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
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
	b := []byte{0xde, 0xad}

	// Wrapped
	ExpectCall(t.wrapped, "Store")(DeepEquals(b)).
		WillOnce(oglemock.Return(blob.Score{}, errors.New("")))

	// Call
	t.store.Store(b)
}

func (t *CheckingStore_StoreTest) WrappedReturnsError() {
	b := []byte{}

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any()).
		WillOnce(oglemock.Return(blob.Score{}, errors.New("taco")))

	// Call
	_, err := t.store.Store(b)

	ExpectThat(err, Error(Equals("taco")))
}

func (t *CheckingStore_StoreTest) WrappedReturnsIncorrectScore() {
	b := []byte{0xde, 0xad}
	correctScore := blob.ComputeScore(b)
	incorrectScore := blob.ComputeScore([]byte{0xbe, 0xef})

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any()).
		WillOnce(oglemock.Return(incorrectScore, nil))

	// Call
	_, err := t.store.Store(b)

	ExpectThat(err, Error(HasSubstr("Incorrect")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(correctScore.Hex())))
	ExpectThat(err, Error(HasSubstr(incorrectScore.Hex())))
}

func (t *CheckingStore_StoreTest) WrappedReturnsCorrectScore() {
	b := []byte{0xde, 0xad}
	correctScore := blob.ComputeScore(b)

	// Wrapped
	ExpectCall(t.wrapped, "Store")(Any()).
		WillOnce(oglemock.Return(correctScore, nil))

	// Call
	score, err := t.store.Store(b)

	AssertEq(nil, err)
	ExpectThat(score, DeepEquals(correctScore))
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type CheckingStore_LoadTest struct {
	checkingStoreTest
}

func init() { RegisterTestSuite(&CheckingStore_LoadTest{}) }

func (t *CheckingStore_LoadTest) CallsWrapped() {
	score := blob.ComputeScore([]byte{0xde, 0xad})

	// Wrapped
	ExpectCall(t.wrapped, "Load")(DeepEquals(score)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(score)
}

func (t *CheckingStore_LoadTest) WrappedReturnsError() {
	score := blob.ComputeScore([]byte{})

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Load(score)

	ExpectThat(err, Error(Equals("taco")))
}

func (t *CheckingStore_LoadTest) WrappedReturnsIncorrectData() {
	correctData := []byte{0xde, 0xad}
	incorrectData := []byte{0xde, 0xad, 0xbe, 0xef}
	score := blob.ComputeScore(correctData)

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any()).
		WillOnce(oglemock.Return(incorrectData, nil))

	// Call
	_, err := t.store.Load(score)

	ExpectThat(err, Error(HasSubstr("Incorrect")))
	ExpectThat(err, Error(HasSubstr("data")))
	ExpectThat(err, Error(HasSubstr(score.Hex())))
}

func (t *CheckingStore_LoadTest) WrappedReturnsCorrectData() {
	correctData := []byte{0xde, 0xad}
	score := blob.ComputeScore(correctData)

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any()).
		WillOnce(oglemock.Return(correctData, nil))

	// Call
	data, err := t.store.Load(score)

	AssertEq(nil, err)
	ExpectThat(data, DeepEquals(correctData))
}

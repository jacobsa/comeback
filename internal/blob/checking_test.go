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
	"context"
	"errors"
	"testing"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestChecking(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type checkingStoreTest struct {
	ctx     context.Context
	wrapped mock_blob.MockStore
	store   blob.Store
}

func (t *checkingStoreTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.wrapped = mock_blob.NewMockStore(ti.MockController, "wrapped")
	t.store = blob.Internal_NewCheckingStore(t.wrapped)
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
	ExpectCall(t.wrapped, "Load")(Any(), DeepEquals(score)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(t.ctx, score)
}

func (t *CheckingStore_LoadTest) WrappedReturnsError() {
	score := blob.ComputeScore([]byte{})

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Load(t.ctx, score)

	ExpectThat(err, Error(Equals("taco")))
}

func (t *CheckingStore_LoadTest) WrappedReturnsIncorrectData() {
	correctData := []byte{0xde, 0xad}
	incorrectData := []byte{0xde, 0xad, 0xbe, 0xef}
	score := blob.ComputeScore(correctData)

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return(incorrectData, nil))

	// Call
	_, err := t.store.Load(t.ctx, score)

	ExpectThat(err, Error(HasSubstr("Incorrect")))
	ExpectThat(err, Error(HasSubstr("data")))
	ExpectThat(err, Error(HasSubstr(score.Hex())))
}

func (t *CheckingStore_LoadTest) WrappedReturnsCorrectData() {
	correctData := []byte{0xde, 0xad}
	score := blob.ComputeScore(correctData)

	// Wrapped
	ExpectCall(t.wrapped, "Load")(Any(), Any()).
		WillOnce(oglemock.Return(correctData, nil))

	// Call
	data, err := t.store.Load(t.ctx, score)

	AssertEq(nil, err)
	ExpectThat(data, DeepEquals(correctData))
}

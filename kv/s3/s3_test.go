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

package s3_test

import (
	"errors"
	"github.com/jacobsa/aws/s3/mock"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/kv/s3"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestS3(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type s3KvStoreTest struct {
	bucket mock_s3.MockBucket
	store  kv.Store
}

func (t *s3KvStoreTest) SetUp(i *TestInfo) {
	t.bucket = mock_s3.NewMockBucket(i.MockController, "bucket")
}

func (t *s3KvStoreTest) createStore() (err error) {
	t.store, err = s3.NewS3KvStore(t.bucket)
	return
}

////////////////////////////////////////////////////////////////////////
// Set
////////////////////////////////////////////////////////////////////////

type SetTest struct {
	s3KvStoreTest

	key string
	val []byte
	err error
}

func init() { RegisterTestSuite(&SetTest{}) }

func (t *SetTest) SetUp(i *TestInfo) {
	// Run common setup code.
	t.s3KvStoreTest.SetUp(i)

	AssertEq(nil, t.createStore())
}

func (t *SetTest) callStore() {
	t.err = t.store.Set([]byte(t.key), t.val)
}

func (t *SetTest) CallsBucketThreeTimes() {
	t.key = "taco"
	t.val = []byte{0xbe, 0xef}

	// StoreObject
	ExpectCall(t.bucket, "StoreObject")("taco", DeepEquals(t.val)).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callStore()
}

func (t *SetTest) BucketReturnsThreeErrors() {
	t.key = "taco"

	// StoreObject
	ExpectCall(t.bucket, "StoreObject")(Any(), Any()).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callStore()

	ExpectNe(nil, t.err)
}

func (t *SetTest) BucketReturnsTwoErrorsThenSucceeds() {
	t.key = "taco"

	// StoreObject
	ExpectCall(t.bucket, "StoreObject")(Any(), Any()).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(errors.New(""))).
		WillOnce(oglemock.Return(nil))

	// Call
	t.callStore()

	ExpectEq(nil, t.err)
}

func (t *SetTest) BucketSucceeds() {
	t.key = "taco"

	// StoreObject
	ExpectCall(t.bucket, "StoreObject")(Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	t.callStore()

	ExpectEq(nil, t.err)
}

////////////////////////////////////////////////////////////////////////
// Get
////////////////////////////////////////////////////////////////////////

type GetTest struct {
	s3KvStoreTest

	key string
	val []byte
	err error
}

func init() { RegisterTestSuite(&GetTest{}) }

func (t *GetTest) SetUp(i *TestInfo) {
	// Run common setup code.
	t.s3KvStoreTest.SetUp(i)

	AssertEq(nil, t.createStore())
}

func (t *GetTest) callStore() {
	t.val, t.err = t.store.Get([]byte(t.key))
}

func (t *GetTest) CallsBucket() {
	t.key = "taco"

	// GetObject
	ExpectCall(t.bucket, "GetObject")("taco").
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callStore()
}

func (t *GetTest) BucketReturnsError() {
	// GetObject
	ExpectCall(t.bucket, "GetObject")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callStore()

	ExpectThat(t.err, Error(HasSubstr("GetObject")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *GetTest) BucketSucceeds() {
	// GetObject
	expected := []byte{0xde, 0xad}
	ExpectCall(t.bucket, "GetObject")(Any()).
		WillOnce(oglemock.Return(expected, nil))

	// Call
	t.callStore()

	AssertEq(nil, t.err)
	ExpectThat(t.val, DeepEquals(expected))
}

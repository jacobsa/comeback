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
	"github.com/jacobsa/aws/s3/mock"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/kv/s3"
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
}

func init() { RegisterTestSuite(&SetTest{}) }

func (t *SetTest) DoesFoo() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Get
////////////////////////////////////////////////////////////////////////

type GetTest struct {
	s3KvStoreTest
}

func init() { RegisterTestSuite(&GetTest{}) }

func (t *GetTest) DoesFoo() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Contains
////////////////////////////////////////////////////////////////////////

type ContainsTest struct {
	s3KvStoreTest
}

func init() { RegisterTestSuite(&ContainsTest{}) }

func (t *ContainsTest) DoesFoo() {
	ExpectEq("TODO", "")
}

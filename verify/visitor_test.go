// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package verify_test

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/verify"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const namePrefix = "blobs/"

type superTest struct {
	ctx       context.Context
	blobStore mock_blob.MockStore
	visitor   graph.Visitor
}

func (t *superTest) setUp(
	ti *TestInfo,
	readFiles bool) {
	t.ctx = ti.Ctx
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")
	t.visitor = verify.NewVisitor(readFiles, nil, t.blobStore)
}

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type CommonTest struct {
	superTest
}

var _ SetUpInterface = &CommonTest{}

func init() { RegisterTestSuite(&CommonTest{}) }

func (t *CommonTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false)
}

func (t *CommonTest) InvalidNodeName() {
	_, err := t.visitor.Visit(t.ctx, "taco")

	ExpectThat(err, Error(HasSubstr("ParseNodeName")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

////////////////////////////////////////////////////////////////////////
// Directories
////////////////////////////////////////////////////////////////////////

type DirsTest struct {
	superTest
}

var _ SetUpInterface = &DirsTest{}

func init() { RegisterTestSuite(&DirsTest{}) }

func (t *DirsTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false)
}

func (t *DirsTest) CallsBlobStore() {
	AssertFalse(true, "TODO")
}

func (t *DirsTest) BlobStoreReaderReturnsError() {
	AssertFalse(true, "TODO")
}

func (t *DirsTest) IncorrectScore() {
	AssertFalse(true, "TODO")
}

func (t *DirsTest) BlobIsJunk() {
	AssertFalse(true, "TODO")
}

func (t *DirsTest) UnknownEntryType() {
	AssertFalse(true, "TODO")
}

func (t *DirsTest) ReturnsAppropriateNodeNames() {
	AssertFalse(true, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Files, no read
////////////////////////////////////////////////////////////////////////

type FilesLiteTest struct {
	superTest
}

var _ SetUpInterface = &FilesLiteTest{}

func init() { RegisterTestSuite(&FilesLiteTest{}) }

func (t *FilesLiteTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false)
}

func (t *FilesLiteTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Files, read
////////////////////////////////////////////////////////////////////////

type FilesFullTest struct {
	superTest
}

var _ SetUpInterface = &FilesFullTest{}

func init() { RegisterTestSuite(&FilesFullTest{}) }

func (t *FilesFullTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false)
}

func (t *FilesFullTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

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

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

const namePrefix = "blobs/"

type commonVisitorTest struct {
	ctx       context.Context
	bucket    gcs.Bucket
	allScores []blob.Score
}

func (t *commonVisitorTest) setUp(
	ti *TestInfo,
	readFiles bool) {
	t.ctx = ti.Ctx
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Directories
////////////////////////////////////////////////////////////////////////

type DirsTest struct {
	commonVisitorTest
}

var _ SetUpInterface = &DirsTest{}

func init() { RegisterTestSuite(&DirsTest{}) }

func (t *DirsTest) SetUp(ti *TestInfo) {
	t.commonVisitorTest.setUp(ti, false)
}

func (t *DirsTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Files, no read
////////////////////////////////////////////////////////////////////////

type FilesLiteTest struct {
	commonVisitorTest
}

var _ SetUpInterface = &FilesLiteTest{}

func init() { RegisterTestSuite(&FilesLiteTest{}) }

func (t *FilesLiteTest) SetUp(ti *TestInfo) {
	t.commonVisitorTest.setUp(ti, false)
}

func (t *FilesLiteTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Files, read
////////////////////////////////////////////////////////////////////////

type FilesFullTest struct {
	commonVisitorTest
}

var _ SetUpInterface = &FilesFullTest{}

func init() { RegisterTestSuite(&FilesFullTest{}) }

func (t *FilesFullTest) SetUp(ti *TestInfo) {
	t.commonVisitorTest.setUp(ti, false)
}

func (t *FilesFullTest) DoesFoo() {
	AssertFalse(true, "TODO")
}

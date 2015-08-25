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

package save

import (
	"testing"

	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
)

func TestBlobStore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx context.Context
}

func init() { RegisterTestSuite(&VisitorTest{}) }

func (t *VisitorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) ScoresAlreadyPresent_Empty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) ScoresAlreadyPresent_NonEmpty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Symlink() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Directory() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_Empty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsFull() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsPartial() {
	AssertTrue(false, "TODO")
}

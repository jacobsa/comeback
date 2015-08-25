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

package graph_test

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestReverseTopsort(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReverseTopsortTreeTest struct {
}

var _ SetUpInterface = &ReverseTopsortTreeTest{}

func init() { RegisterTestSuite(&ReverseTopsortTreeTest{}) }

func (t *ReverseTopsortTreeTest) SetUp(ti *TestInfo) {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReverseTopsortTreeTest) Empty() {
	AssertTrue(false, "TODO")
}

func (t *ReverseTopsortTreeTest) NoBranching() {
	AssertTrue(false, "TODO")
}

func (t *ReverseTopsortTreeTest) LittleBranching() {
	AssertTrue(false, "TODO")
}

func (t *ReverseTopsortTreeTest) LotsOfBranching() {
	AssertTrue(false, "TODO")
}

func (t *ReverseTopsortTreeTest) LargeTree() {
	AssertTrue(false, "TODO")
}

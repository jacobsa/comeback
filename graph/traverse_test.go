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

func TestTraverse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type TraverseTest struct {
}

var _ SetUpTestSuiteInterface = &TraverseTest{}
var _ TearDownTestSuiteInterface = &TraverseTest{}

func init() { RegisterTestSuite(&TraverseTest{}) }

func (t *TraverseTest) SetUpTestSuite() {
	panic("TODO: GOMAXPROCS")
}

func (t *TraverseTest) TearDownTestSuite() {
	panic("TODO: GOMAXPROCS")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TraverseTest) EmptyGraph() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) SimpleRootedTree() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) SimpleDAG() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) MultipleConnectedComponents() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) LargeRootedTree() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) VisitorReturnsError() {
	AssertFalse(true, "TODO")
}

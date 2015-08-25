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

	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
)

func TestTraverseDAG(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const traverseDAGParallelism = 8

type TraverseDAGTest struct {
	ctx context.Context
}

var _ SetUpInterface = &TraverseDAGTest{}

func init() { RegisterTestSuite(&TraverseDAGTest{}) }

func (t *TraverseDAGTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TraverseDAGTest) EmptyGraph() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) SingleNodeConnectedComponents() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) SimpleRootedTree() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) SimpleDAG() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) MultipleConnectedComponents() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) LargeRootedTree() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) LargeRootedTree_Inverted() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) SuccessorFinderReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *TraverseDAGTest) VisitorReturnsError() {
	AssertTrue(false, "TODO")
}

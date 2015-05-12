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
	"runtime"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/graph"
	. "github.com/jacobsa/ogletest"
)

func TestTraverse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helper
////////////////////////////////////////////////////////////////////////

// A graph.Visitor that invokes a wrapped function.
type funcVisitor struct {
	F func(context.Context, string) ([]string, error)
}

var _ graph.Visitor = &funcVisitor{}

func (fv *funcVisitor) Visit(
	ctx context.Context,
	node string) (adjacent []string, err error) {
	adjacent, err = fv.F(ctx, node)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const parallelism = 16

type TraverseTest struct {
	ctx context.Context
}

var _ SetUpTestSuiteInterface = &TraverseTest{}
var _ SetUpInterface = &TraverseTest{}

func init() { RegisterTestSuite(&TraverseTest{}) }

func (t *TraverseTest) SetUpTestSuite() {
	// Ensure that we get real parallelism where available.
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (t *TraverseTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TraverseTest) EmptyGraph() {
	// Visitor -- should never be called.
	f := func(ctx context.Context, node string) (adjacent []string, err error) {
		AddFailure("Visitor unexpectedly called with node %q", node)
		return
	}

	v := &funcVisitor{F: f}

	// Traverse.
	roots := []string{}
	err := graph.Traverse(t.ctx, parallelism, roots, v)

	AssertEq(nil, err)
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

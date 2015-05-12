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
	"sort"
	"sync"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/graph"
	. "github.com/jacobsa/oglematchers"
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

func sortNodes(in []string) (out sort.StringSlice) {
	out = make(sort.StringSlice, len(in))
	copy(out, in)
	sort.Sort(out)
	return
}

func indexNodes(nodes []string) (index map[string]int) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const parallelism = 16

type TraverseTest struct {
	ctx context.Context
	mu  sync.Mutex

	// The roots from which we traverse.
	roots []string

	// Edges used by the default implementation of visit.
	edges map[string][]string

	// The function that will be called for visiting nodes. By default, this
	// writes to nodesVisited and reads from edges.
	visit func(context.Context, string) ([]string, error)

	// The nodes that were visited, in the order in which they were visited.
	//
	// GUARDED_BY(mu)
	nodesVisited []string
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

func (t *TraverseTest) defaultVisit(
	ctx context.Context,
	node string) (adjacent []string, err error) {
	t.mu.Lock()
	t.nodesVisited = append(t.nodesVisited, node)
	t.mu.Unlock()

	adjacent = t.edges[node]
	return
}

func (t *TraverseTest) traverse() (err error) {
	// Choose a visit function.
	visit := t.visit
	if visit == nil {
		visit = t.defaultVisit
	}

	// Traverse.
	v := &funcVisitor{F: visit}
	err = graph.Traverse(t.ctx, parallelism, t.roots, v)

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TraverseTest) EmptyGraph() {
	// Traverse.
	err := t.traverse()
	AssertEq(nil, err)

	// Nothing should have been visited.
	ExpectEq(0, len(t.nodesVisited))
}

func (t *TraverseTest) SingleNodeConnectedComponents() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) SimpleRootedTree() {
	// Graph structure:
	//
	//        A
	//      / |
	//     B  C
	//      / | \
	//     D  E  F
	//      / |  |
	//     G  H  I
	//           | \
	//           J  K
	//
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"C": []string{"D", "E", "F"},
		"E": []string{"G", "H"},
		"F": []string{"I"},
		"I": []string{"J", "K"},
	}

	// Traverse.
	err := t.traverse()
	AssertEq(nil, err)

	AssertThat(
		sortNodes(t.nodesVisited),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
			"E",
			"F",
			"G",
			"H",
			"I",
			"J",
			"K",
		))

	nodeIndex := indexNodes(t.nodesVisited)
	for p, successors := range t.edges {
		for _, s := range successors {
			ExpectLt(nodeIndex[p], nodeIndex[s], "%q %q", p, s)
		}
	}
}

func (t *TraverseTest) SimpleDAG() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) MultipleConnectedComponents() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) RedundantRoots() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) Cycle() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) LargeRootedTree() {
	AssertFalse(true, "TODO")
}

func (t *TraverseTest) VisitorReturnsError() {
	AssertFalse(true, "TODO")
}

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
	"fmt"
	"math/rand"
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
	index = make(map[string]int)
	for i, n := range nodes {
		index[n] = i
	}

	return
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
	t.edges = make(map[string][]string)
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
	// Graph structure:
	//
	//     A  B  C  D
	//
	t.roots = []string{"A", "B", "C", "D"}
	t.edges = map[string][]string{}

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
		))
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
	t.roots = []string{"A"}
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
	// Graph structure:
	//
	//        A
	//      /  \
	//     B    C
	//      \  /|
	//        D |
	//         \|
	//          E
	//
	t.roots = []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D", "E"},
		"D": []string{"E"},
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
		))

	nodeIndex := indexNodes(t.nodesVisited)
	ExpectGt(nodeIndex["B"], nodeIndex["A"])
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["D"],
		AnyOf(GreaterThan(nodeIndex["B"]), GreaterThan(nodeIndex["C"])))

	ExpectThat(
		nodeIndex["E"],
		AnyOf(GreaterThan(nodeIndex["D"]), GreaterThan(nodeIndex["C"])))
}

func (t *TraverseTest) MultipleConnectedComponents() {
	// Graph structure:
	//
	//        A       E
	//      /  \      |\
	//     B    C     F G
	//      \  /
	//        D
	//
	t.roots = []string{"A", "E"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D"},
		"E": []string{"F", "G"},
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
		))

	nodeIndex := indexNodes(t.nodesVisited)
	ExpectGt(nodeIndex["B"], nodeIndex["A"])
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["D"],
		AnyOf(GreaterThan(nodeIndex["B"]), GreaterThan(nodeIndex["C"])))

	ExpectGt(nodeIndex["F"], nodeIndex["E"])
	ExpectGt(nodeIndex["G"], nodeIndex["E"])
}

func (t *TraverseTest) RedundantRoots() {
	// Graph structure:
	//
	//        A
	//      /  \
	//     B    C
	//      \  /|
	//        D |
	//         \|
	//          E
	//
	t.roots = []string{"A", "D", "B"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D", "E"},
		"D": []string{"E"},
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
		))

	nodeIndex := indexNodes(t.nodesVisited)
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["E"],
		AnyOf(GreaterThan(nodeIndex["D"]), GreaterThan(nodeIndex["C"])))
}

func (t *TraverseTest) Cycle() {
	// Graph structure:
	//
	//        A
	//      / ^\
	//     B  | C
	//      \ |/
	//        D
	//
	t.roots = []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D"},
		"D": []string{"A"},
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
		))
}

func (t *TraverseTest) LargeRootedTree() {
	// Set up a tree of the given depth, with a random number of children for
	// each node.
	const depth = 10
	t.roots = []string{"root"}

	nextID := 0
	nextLevel := []string{"root"}
	allNodes := map[string]struct{}{
		"root": struct{}{},
	}

	for depthI := 0; depthI < depth; depthI++ {
		thisLevel := nextLevel
		nextLevel = nil
		for _, parent := range thisLevel {
			numChildren := int(rand.Int31n(6))
			for childI := 0; childI < numChildren; childI++ {
				child := fmt.Sprintf("%v", nextID)
				nextID++

				nextLevel = append(nextLevel, child)
				t.edges[parent] = append(t.edges[parent], child)
				allNodes[child] = struct{}{}
			}
		}
	}

	// Traverse.
	err := t.traverse()
	AssertEq(nil, err)

	// All ndoes should be represented.
	AssertEq(len(allNodes), len(t.nodesVisited))
	for _, n := range t.nodesVisited {
		_, ok := allNodes[n]
		AssertTrue(ok, "Unexpected node: %q", n)
	}

	// Edge order should be respected.
	nodeIndex := indexNodes(t.nodesVisited)
	for p, successors := range t.edges {
		for _, s := range successors {
			ExpectLt(nodeIndex[p], nodeIndex[s], "%q %q", p, s)
		}
	}
}

func (t *TraverseTest) VisitorReturnsError() {
	AssertFalse(true, "TODO")
}

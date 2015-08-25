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
	"errors"
	"sort"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
)

func TestExplore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// A graph.SuccessorFinder that invokes a wrapped function, and assumes that
// all nodes are strings.
type successorFinder struct {
	F func(context.Context, string) ([]string, error)
}

var _ graph.SuccessorFinder = &successorFinder{}

func (sf *successorFinder) FindDirectSuccessors(
	ctx context.Context,
	node graph.Node) (successors []graph.Node, err error) {
	var a []string
	a, err = sf.F(ctx, node.(string))

	for _, a := range a {
		successors = append(successors, a)
	}

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

const exploreParallelism = 16

type ExploreDirectedGraphTest struct {
	ctx context.Context

	// Edges used by the default implementation of findDirectSuccessors.
	edges map[string][]string

	// The function that will be called for finding direct successors. By
	// default, this reads from edges.
	findDirectSuccessors func(context.Context, string) ([]string, error)
}

var _ SetUpInterface = &ExploreDirectedGraphTest{}

func init() { RegisterTestSuite(&ExploreDirectedGraphTest{}) }

func (t *ExploreDirectedGraphTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.edges = make(map[string][]string)
}

func (t *ExploreDirectedGraphTest) defaultFindDirectSuccessors(
	ctx context.Context,
	node string) (successors []string, err error) {
	successors = t.edges[node]
	return
}

func (t *ExploreDirectedGraphTest) explore(
	roots []string) (nodes []string, err error) {
	// Choose a "find direct successors" function.
	findDirectSuccessors := t.findDirectSuccessors
	if findDirectSuccessors == nil {
		findDirectSuccessors = t.defaultFindDirectSuccessors
	}

	// Convert to a list of nodes.
	var rootNodes []graph.Node
	for _, n := range roots {
		rootNodes = append(rootNodes, n)
	}

	b := syncutil.NewBundle(t.ctx)

	// Write nodes into a channel.
	nodeChan := make(chan graph.Node)
	b.Add(func(ctx context.Context) (err error) {
		defer close(nodeChan)

		sf := &successorFinder{F: findDirectSuccessors}
		err = graph.ExploreDirectedGraph(
			ctx,
			sf,
			rootNodes,
			nodeChan,
			exploreParallelism)

		return
	})

	// Accumulate into the output slice.
	b.Add(func(ctx context.Context) (err error) {
		for n := range nodeChan {
			nodes = append(nodes, n.(string))
		}

		return
	})

	err = b.Join()
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ExploreDirectedGraphTest) EmptyGraph() {
	roots := []string{}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	// Nothing should have been emitted.
	ExpectEq(0, len(nodes))
}

func (t *ExploreDirectedGraphTest) SingleNodeConnectedComponents() {
	// Graph structure:
	//
	//     A  B  C  D
	//
	roots := []string{"A", "B", "C", "D"}
	t.edges = map[string][]string{}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
		))
}

func (t *ExploreDirectedGraphTest) SimpleRootedTree() {
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
	roots := []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"C": []string{"D", "E", "F"},
		"E": []string{"G", "H"},
		"F": []string{"I"},
		"I": []string{"J", "K"},
	}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
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

	nodeIndex := indexNodes(nodes)
	for p, successors := range t.edges {
		for _, s := range successors {
			ExpectLt(nodeIndex[p], nodeIndex[s], "%q %q", p, s)
		}
	}
}

func (t *ExploreDirectedGraphTest) SimpleDAG() {
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
	roots := []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D", "E"},
		"D": []string{"E"},
	}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
			"E",
		))

	nodeIndex := indexNodes(nodes)
	ExpectGt(nodeIndex["B"], nodeIndex["A"])
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["D"],
		AnyOf(GreaterThan(nodeIndex["B"]), GreaterThan(nodeIndex["C"])))

	ExpectThat(
		nodeIndex["E"],
		AnyOf(GreaterThan(nodeIndex["D"]), GreaterThan(nodeIndex["C"])))
}

func (t *ExploreDirectedGraphTest) MultipleConnectedComponents() {
	// Graph structure:
	//
	//        A       E
	//      /  \      |\
	//     B    C     F G
	//      \  /
	//        D
	//
	roots := []string{"A", "E"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D"},
		"E": []string{"F", "G"},
	}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
			"E",
			"F",
			"G",
		))

	nodeIndex := indexNodes(nodes)
	ExpectGt(nodeIndex["B"], nodeIndex["A"])
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["D"],
		AnyOf(GreaterThan(nodeIndex["B"]), GreaterThan(nodeIndex["C"])))

	ExpectGt(nodeIndex["F"], nodeIndex["E"])
	ExpectGt(nodeIndex["G"], nodeIndex["E"])
}

func (t *ExploreDirectedGraphTest) RedundantRoots() {
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
	roots := []string{"A", "D", "B", "A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D", "E"},
		"D": []string{"E"},
	}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
			"E",
		))

	nodeIndex := indexNodes(nodes)
	ExpectGt(nodeIndex["C"], nodeIndex["A"])

	ExpectThat(
		nodeIndex["E"],
		AnyOf(GreaterThan(nodeIndex["D"]), GreaterThan(nodeIndex["C"])))
}

func (t *ExploreDirectedGraphTest) Cycle() {
	// Graph structure:
	//
	//        A
	//      / ^\
	//     B  | C
	//      \ |/
	//        D
	//
	roots := []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"D"},
		"D": []string{"A"},
	}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
			"D",
		))
}

func (t *ExploreDirectedGraphTest) LargeRootedTree() {
	const depth = 10
	t.edges = randomTree(depth)
	roots := []string{"root"}

	// Explore.
	nodes, err := t.explore(roots)
	AssertEq(nil, err)

	// All nodes should be represented.
	AssertEq(len(t.edges), len(nodes))
	for _, n := range nodes {
		_, ok := t.edges[n]
		AssertTrue(ok, "Unexpected node: %q", n)
	}

	// Edge order should be respected.
	nodeIndex := indexNodes(nodes)
	for p, successors := range t.edges {
		for _, s := range successors {
			ExpectLt(nodeIndex[p], nodeIndex[s], "%q %q", p, s)
		}
	}
}

func (t *ExploreDirectedGraphTest) VisitorReturnsError() {
	AssertGt(exploreParallelism, 1)

	// Graph structure:
	//
	//        A
	//      / |
	//     B  C
	//     |  | \
	//     D  E  F
	//      / |  |
	//     G  H  I
	//           | \
	//           J  K
	//
	roots := []string{"A"}
	t.edges = map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D"},
		"C": []string{"E", "F"},
		"E": []string{"G", "H"},
		"F": []string{"I"},
		"I": []string{"J", "K"},
	}

	// Operate as normal, except:
	//
	//  *  For C, wait until told and then return an error.
	//
	//  *  For B:
	//     *   Tell C to proceed.
	//     *   Block until cancelled.
	//     *   Close a channel indicating that the context was cancelled.
	//     *   Return children as normal.
	//
	cErr := errors.New("taco")
	bReceived := make(chan struct{})
	bCancelled := make(chan struct{})

	t.findDirectSuccessors = func(
		ctx context.Context,
		node string) (successors []string, err error) {
		// Call through.
		successors, err = t.defaultFindDirectSuccessors(ctx, node)

		// Perform special behavior.
		switch node {
		case "C":
			<-bReceived
			err = cErr

		case "B":
			close(bReceived)

			done := ctx.Done()
			AssertNe(nil, done)
			<-done
			close(bCancelled)
		}

		return
	}

	// Explore.
	nodes, err := t.explore(roots)
	ExpectEq(cErr, err)

	// B should have seen cancellation.
	<-bCancelled

	// Nothing descending from B or C should have been visted.
	ExpectThat(
		sortNodes(nodes),
		ElementsAre("A", "B", "C"))
}

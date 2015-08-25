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
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestTraverseDAG(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Return a topological sort of the nodes of the DAG defined by the supplied
// edge map.
func topsort(edges map[string][]string) (nodes []string, err error) {
	// A DFS-based algorithm that detects cycles.
	// Cf. https://en.wikipedia.org/wiki/Topological_sorting#Algorithms
	marked := make(map[string]struct{})
	tempMarked := make(map[string]struct{})
	unmarked := make(map[string]struct{})
	for n, _ := range edges {
		unmarked[n] = struct{}{}
	}

	var visit func(string) error
	visit = func(n string) (err error) {
		// Cycle?
		if _, ok := tempMarked[n]; ok {
			err = fmt.Errorf("Cycle containing %q detected", n)
			return
		}

		// Already visited?
		if _, ok := marked[n]; ok {
			return
		}

		delete(unmarked, n)
		tempMarked[n] = struct{}{}
		for _, m := range edges[n] {
			err = visit(m)
			if err != nil {
				return
			}
		}

		marked[n] = struct{}{}
		delete(tempMarked, n)
		nodes = append([]string{n}, nodes...)

		return
	}

	// Visit nodes until there are no unmarked ones left.
	for len(unmarked) > 0 {
		var someNode string
		for n, _ := range unmarked {
			someNode = n
			break
		}

		err = visit(someNode)
		if err != nil {
			return
		}
	}

	return
}

// Compute the the reachability partial order for the DAG defined by the
// supplied edges. Don't include self-reachability.
func reachabilityRelation(
	edges map[string][]string) (r map[string][]string, err error) {
	r = make(map[string][]string)

	var visit func(string)
	visit = func(n string) {
		// Have we already computed the result for n?
		if _, ok := r[n]; ok {
			return
		}

		// The set of things reachable from n is the set of things reachable from
		// its successors, plus those successors.
		set := make(map[string]struct{})
		for _, successor := range edges[n] {
			set[successor] = struct{}{}

			visit(successor)
			for _, m := range r[successor] {
				set[m] = struct{}{}
			}
		}

		for m, _ := range set {
			r[n] = append(r[n], m)
		}
	}

	for n, _ := range edges {
		visit(n)
	}

	return
}

// Return the relation composed of pairs (Y, X) for each pair (X, Y) in the
// input relation.
func invertRelation(r map[string][]string) (inverted map[string][]string) {
	inverted = make(map[string][]string)
	for k, vs := range r {
		for _, v := range vs {
			inverted[v] = append(inverted[v], k)
		}
	}

	// Ensure that everything has an entry, even if empty.
	for k, _ := range r {
		if _, ok := inverted[k]; !ok {
			inverted[k] = nil
		}
	}

	return
}

// Create a rand.Rand seeded with a good source.
func makeRandSource() (src *rand.Rand) {
	// Read a seed from a good source.
	var seed int64
	err := binary.Read(cryptorand.Reader, binary.LittleEndian, &seed)
	if err != nil {
		log.Fatalln(err)
	}

	src = rand.New(rand.NewSource(seed))
	return
}

// Create a SuccessorFinder that consults the supplied map of edges.
func successorFinderForEdges(
	edges map[string][]string) (sf graph.SuccessorFinder) {
	sf = &successorFinder{
		F: func(ctx context.Context, n string) (nodes []string, err error) {
			nodes = edges[n]
			return
		},
	}

	return
}

// A graph.Visitor for string nodes that calls through to a canned function.
type visitor struct {
	F func(context.Context, string) error
}

func (v *visitor) Visit(
	ctx context.Context,
	untyped graph.Node) (err error) {
	err = v.F(ctx, untyped.(string))
	return
}

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

func (t *TraverseDAGTest) call(
	edges map[string][]string,
	visit func(context.Context, string) error) (err error) {
	// Sort the nodes into a valid order and stick them in a channel.
	nodes, err := topsort(edges)
	if err != nil {
		err = fmt.Errorf("topsort: %v", err)
		return
	}

	nodeChan := make(chan graph.Node, len(nodes))
	for _, n := range nodes {
		nodeChan <- n
	}
	close(nodeChan)

	// Call.
	err = graph.TraverseDAG(
		t.ctx,
		nodeChan,
		successorFinderForEdges(edges),
		&visitor{F: visit},
		traverseDAGParallelism)

	return
}

// Run the graph described by the given edges through TraverseDAG, inserting
// sleeps to attempt to shake out races, and make sure we can't catch it
// violating any of its promises.
func (t *TraverseDAGTest) runTest(edges map[string][]string) {
	// Compute the reachability relation for the DAG, and invert it.
	reachable, err := reachabilityRelation(edges)
	AssertEq(nil, err)
	inverseReachable := invertRelation(reachable)

	// Set up a visit function that ensures that a visit hasn't happened out of
	// order, sleeps for awhile, then marks the node as visited.
	randSrc := makeRandSource()

	var mu sync.Mutex
	visited := make(map[string]struct{}) // GUARDED_BY(mu)

	visit := func(ctx context.Context, n string) (err error) {
		mu.Lock()
		defer mu.Unlock()

		// We should have already finished visiting all of the nodes from which
		// this node is reachable.
		for _, p := range inverseReachable[n] {
			_, ok := visited[p]
			if !ok {
				err = fmt.Errorf("Visited %q before finished visiting %q", n, p)
				return
			}
		}

		// Wait a random amount of time, then mark the node as visited and return.
		mu.Unlock()
		time.Sleep(time.Duration(randSrc.Int63n(int64(20 * time.Millisecond))))
		mu.Lock()

		visited[n] = struct{}{}
		return
	}

	// Call.
	err = t.call(edges, visit)
	AssertEq(nil, err)

	// Make sure everything was visited.
	for n, _ := range edges {
		_, ok := visited[n]
		AssertTrue(ok, "n: %q", n)
	}
}

// Generate a tree with a certain depth, where the number of children for each
// node is random. The root node is "root".
func randomTree(depth int) (edges map[string][]string) {
	edges = make(map[string][]string)
	randSrc := makeRandSource()

	nextID := 0
	nextLevel := []string{"root"}

	for depthI := 0; depthI < depth; depthI++ {
		thisLevel := nextLevel
		nextLevel = nil

		for _, parent := range thisLevel {
			// Ensure that there is an entry, even if there are no children.
			edges[parent] = []string{}

			// Add children.
			numChildren := int(randSrc.Int31n(6))
			for childI := 0; childI < numChildren; childI++ {
				child := fmt.Sprintf("%v", nextID)
				nextID++

				nextLevel = append(nextLevel, child)
				edges[parent] = append(edges[parent], child)
			}
		}
	}

	// Ensure that there is an entry for everything in the last level, even if
	// there are no children.
	for _, n := range nextLevel {
		edges[n] = []string{}
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *TraverseDAGTest) EmptyGraph() {
	edges := map[string][]string{}
	t.runTest(edges)
}

func (t *TraverseDAGTest) SingleNodeConnectedComponents() {
	// Graph structure:
	//
	//     A  B  C  D
	//
	edges := map[string][]string{
		"A": {},
		"B": {},
		"C": {},
		"D": {},
	}

	t.runTest(edges)
}

func (t *TraverseDAGTest) SimpleRootedTree() {
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
	edges := map[string][]string{
		"A": {"B", "C"},
		"B": {},
		"C": {"D", "E", "F"},
		"D": {},
		"E": {"G", "H"},
		"F": {"I"},
		"G": {},
		"H": {},
		"I": {"J", "K"},
		"J": {},
		"K": {},
	}

	t.runTest(edges)
}

func (t *TraverseDAGTest) SimpleDAG() {
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
	edges := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D", "E"},
		"D": {"E"},
		"E": {},
	}

	t.runTest(edges)
}

func (t *TraverseDAGTest) MultipleConnectedComponents() {
	// Graph structure:
	//
	//        A       E
	//      /  \      |\
	//     B    C     F G
	//      \  /
	//        D
	//
	edges := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D"},
		"D": {},
		"E": {"F", "G"},
		"F": {},
		"G": {},
	}

	t.runTest(edges)
}

func (t *TraverseDAGTest) LargeRootedTree() {
	const depth = 6
	edges := randomTree(depth)
	t.runTest(edges)
}

func (t *TraverseDAGTest) LargeRootedTree_Inverted() {
	const depth = 6
	edges := invertRelation(randomTree(depth))
	t.runTest(edges)
}

func (t *TraverseDAGTest) VisitorReturnsError() {
	AssertGt(traverseDAGParallelism, 1)

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
	edges := map[string][]string{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"E", "F"},
		"D": {},
		"E": {"G", "H"},
		"F": {"I"},
		"G": {},
		"H": {},
		"I": {"J", "K"},
		"J": {},
		"K": {},
	}

	// Visitor behavior:
	//
	//  *  Record the node visited.
	//
	//  *  For C, wait until told and then return an error.
	//
	//  *  For B:
	//     *   Tell C to proceed.
	//     *   Block until cancelled.
	//     *   Close a channel indicating that the context was cancelled.
	//
	cErr := errors.New("taco")
	bReceived := make(chan struct{})
	bCancelled := make(chan struct{})
	visited := make(chan string, len(edges))

	visit := func(ctx context.Context, n string) (err error) {
		visited <- n

		switch n {
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

	// Traverse.
	err := t.call(edges, visit)
	ExpectEq(cErr, err)

	// B should have seen cancellation.
	<-bCancelled

	// Nothing descending from B or C should have been visted.
	close(visited)
	var nodes []string
	for n := range visited {
		nodes = append(nodes, n)
	}

	ExpectThat(sortNodes(nodes), ElementsAre("A", "B", "C"))
}

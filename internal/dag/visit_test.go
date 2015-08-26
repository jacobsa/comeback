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

package dag_test

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/dag"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
)

func TestVisit(t *testing.T) { RunTests(t) }

var fCheckInvariants = flag.Bool(
	"check_invariants",
	false,
	"Enable syncutil.InvariantMutex checking.")

func TestMain(m *testing.M) {
	flag.Parse()

	if *fCheckInvariants {
		syncutil.EnableInvariantChecking()
	}

	os.Exit(m.Run())
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// A dag.DependencyResolver that invokes a wrapped function, and assumes that
// all nodes are strings.
type dependencyResolver struct {
	F func(context.Context, string) ([]string, error)
}

var _ dag.DependencyResolver = &dependencyResolver{}

func (dr *dependencyResolver) FindDependencies(
	ctx context.Context,
	node dag.Node) (deps []dag.Node, err error) {
	var a []string
	a, err = dr.F(ctx, node.(string))

	for _, a := range a {
		deps = append(deps, a)
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

// A dag.Visitor for string nodes that calls through to a canned function.
type visitor struct {
	F func(context.Context, string) error
}

func (v *visitor) Visit(
	ctx context.Context,
	untyped dag.Node) (err error) {
	err = v.F(ctx, untyped.(string))
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const visitParallelism = 8

type VisitTest struct {
	ctx context.Context
}

var _ SetUpInterface = &VisitTest{}

func init() { RegisterTestSuite(&VisitTest{}) }

func (t *VisitTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
}

func (t *VisitTest) call(
	startNodes []string,
	findDependencies func(context.Context, string) ([]string, error),
	visit func(context.Context, string) error) (err error) {
	// Convert to a list of dag.Node.
	var startDAGNodes []dag.Node
	for _, n := range startNodes {
		startDAGNodes = append(startDAGNodes, n)
	}

	// Call.
	err = dag.Visit(
		t.ctx,
		startDAGNodes,
		&dependencyResolver{F: findDependencies},
		&visitor{F: visit},
		visitParallelism)

	return
}

// Run the graph described by the given edges through Visit, inserting
// sleeps to attempt to shake out races, and make sure we can't catch it
// violating any of its promises.
func (t *VisitTest) runTest(
	edges map[string][]string,
	startNodes []string) {
	// Compute the reachability relation for the DAG, and invert it.
	reachable, err := reachabilityRelation(edges)
	AssertEq(nil, err)
	inverseReachable := invertRelation(reachable)

	// Set up a visit function that ensures that a visit hasn't happened out of
	// order, sleeps for awhile, then marks the node as visited.
	randSrc := makeRandSource()

	var mu sync.Mutex
	resolved := make(map[string]struct{}) // GUARDED_BY(mu)
	visited := make(map[string]struct{})  // GUARDED_BY(mu)

	visit := func(ctx context.Context, n string) (err error) {
		mu.Lock()
		defer mu.Unlock()

		// We should have already finished finding dependencies for this node.
		if _, ok := resolved[n]; !ok {
			err = fmt.Errorf("Visited %q before finished resolving it", n)
			return
		}

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

		// This node should not yet have been visited.
		if _, ok := visited[n]; ok {
			err = fmt.Errorf("Node %q visited twice", n)
			return
		}

		visited[n] = struct{}{}
		return
	}

	// Set up a similar dependency resolution function.
	findDependencies := func(
		ctx context.Context,
		n string) (deps []string, err error) {
		mu.Lock()
		defer mu.Unlock()

		// Wait a random amount of time, then mark the node as resolved and return
		// the appropriate dependencies.
		mu.Unlock()
		time.Sleep(time.Duration(randSrc.Int63n(int64(20 * time.Millisecond))))
		mu.Lock()

		// This node should not yet have been resolved.
		if _, ok := resolved[n]; ok {
			err = fmt.Errorf("Node %q resolved twice", n)
			return
		}

		resolved[n] = struct{}{}
		deps = edges[n]

		return
	}

	// Call.
	err = t.call(startNodes, findDependencies, visit)
	AssertEq(nil, err)

	// Make sure everything was resolved and visited.
	for n, _ := range edges {
		var ok bool

		_, ok = resolved[n]
		AssertTrue(ok, "Not resolved: %q", n)

		_, ok = visited[n]
		AssertTrue(ok, "Not visited: %q", n)
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

func (t *VisitTest) EmptyGraph() {
	edges := map[string][]string{}
	startNodes := []string{}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) SingleNodeConnectedComponents() {
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
	startNodes := []string{"A", "B", "C", "D"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) SimpleRootedTree() {
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
	startNodes := []string{"A"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) SimpleDAG() {
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
	startNodes := []string{"A"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) MultipleConnectedComponents() {
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
	startNodes := []string{"A", "E"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) RedundantRoots() {
	AssertTrue(false, "TODO")
}

func (t *VisitTest) Cycle() {
	AssertTrue(false, "TODO")
}

func (t *VisitTest) LargeRootedTree() {
	const depth = 6
	edges := randomTree(depth)
	startNodes := []string{"root"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) LargeRootedTree_Inverted() {
	const depth = 6
	edges := invertRelation(randomTree(depth))
	startNodes := []string{"root"}

	t.runTest(edges, startNodes)
}

func (t *VisitTest) DependencyResolverReturnsError() {
	AssertTrue(false, "TODO")
}

func (t *VisitTest) VisitorReturnsError() {
	AssertGt(visitParallelism, 1)

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
	startNodes := []string{"A"}

	// Dependency finder: operate as usual.
	findDependencies := func(
		ctx context.Context,
		n string) (deps []string, err error) {
		deps = edges[n]
		return
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

	// Call.
	err := t.call(startNodes, findDependencies, visit)
	ExpectEq(cErr, err)

	// B should have seen cancellation.
	<-bCancelled

	// Nothing descending from B or C should have been visted.
	close(visited)
	var nodes sort.StringSlice
	for n := range visited {
		nodes = append(nodes, n)
	}

	sort.Sort(nodes)
	ExpectThat(nodes, ElementsAre("A", "B", "C"))
}

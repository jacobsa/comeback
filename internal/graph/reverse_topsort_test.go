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
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestReverseTopsort(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ReverseTopsortTreeTest struct {
	ctx context.Context
}

var _ SetUpInterface = &ReverseTopsortTreeTest{}

func init() { RegisterTestSuite(&ReverseTopsortTreeTest{}) }

func (t *ReverseTopsortTreeTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
}

func (t *ReverseTopsortTreeTest) run(
	root string,
	edges map[string][]string) (nodes []string, err error) {
	// Set up a successor finder.
	sf := &successorFinder{
		F: func(ctx context.Context, n string) (successors []string, err error) {
			successors = edges[n]
			return
		},
	}

	// Call through.
	c := make(chan graph.Node, 10e3)
	err = graph.ReverseTopsortTree(
		t.ctx,
		sf,
		root,
		c)

	if err != nil {
		err = fmt.Errorf("ReverseTopsortTree: %v", err)
		return
	}

	// Convert the nodes returned in the channel.
	close(c)
	for n := range c {
		nodes = append(nodes, n.(string))
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ReverseTopsortTreeTest) SingleNode() {
	// Graph structure:
	//
	//        A
	//
	root := "A"
	edges := map[string][]string{}

	// Call
	nodes, err := t.run(root, edges)
	AssertEq(nil, err)

	ExpectThat(nodes, ElementsAre("A"))
}

func (t *ReverseTopsortTreeTest) NoBranching() {
	// Graph structure:
	//
	//        A
	//        |
	//        B
	//        |
	//        C
	//
	root := "A"
	edges := map[string][]string{
		"A": {"B"},
		"B": {"C"},
	}

	// Call
	nodes, err := t.run(root, edges)
	AssertEq(nil, err)

	AssertThat(
		sortNodes(nodes),
		ElementsAre(
			"A",
			"B",
			"C",
		))

	nodeIndex := indexNodes(nodes)
	for p, successors := range edges {
		for _, s := range successors {
			ExpectLt(nodeIndex[s], nodeIndex[p], "%q -> %q", p, s)
		}
	}
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

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

package graph

import (
	"errors"
	"log"

	"github.com/jacobsa/syncutil"

	"golang.org/x/net/context"
)

// Given
//
// *   a channel whose contents are a topologically-sorted list of the unique
//     nodes in a DAG (i.e. a node appears only after all of its predecessors
//     have) and
//
// *   a successor finder that agrees with the topological sort about the
//     structure of the graph,
//
// invoke the supplied visitor once for each node in the graph with, bounded
// parallelism. The visitor will be called for a node N only after it has
// returned success for all of N's predecessors.
func TraverseDAG(
	ctx context.Context,
	nodes <-chan Node,
	sf SuccessorFinder,
	v Visitor,
	parallelism int) (err error) {
	err = errors.New("TODO")
	return
}

type traverseDagState struct {
	mu syncutil.InvariantMutex

	// A map containing nodes that we are not yet ready to visit, because they
	// have predecessors that have not yet been visited or because we have not
	// yet seen them.
	//
	// INVARIANT: For all v, !v.readyToVisit()
	// INVARIANT: For all v, v.checkInvariants doesn't panic
	//
	// GUARDED_BY(mu)
	notReadyToVisit map[Node]traverseDagNodeState

	// A list of nodes that we are ready to visit but have not yet started
	// visiting.
	//
	// INVARIANT: For all n, n is not a key in notReadyToVisit
	//
	// GUARDED_BY(mu)
	readyToVisit []Node

	// A channel of nodes that are ready to visit. Visitor workers sleep on this
	// channel when readyToVisit is empty. To avoid deadlock, it is never written
	// to by visitor workers; only the driver that processes incoming nodes from
	// the user. Similarly, the latter never writes to readyToVisit, because its
	// updates may be missed by the former.
	newReadyNodes chan Node
}

// LOCKS_REQUIRED(s.mu)
func (s *traverseDagState) checkInvariants() {
	// INVARIANT: For all v, !v.readyToVisit()
	for k, v := range s.notReadyToVisit {
		if v.readyToVisit() {
			log.Panicf("Unexpected ready node: %#v, %#v", k, v)
		}
	}

	// INVARIANT: For all v, v.checkInvariants doesn't panic
	for _, v := range s.notReadyToVisit {
		v.checkInvariants()
	}

	// INVARIANT: For all n, n is not a key in notReadyToVisit
	for _, n := range s.readyToVisit {
		if _, ok := s.notReadyToVisit[n]; ok {
			log.Panicf("Ready and not ready: %#v", n)
		}
	}
}

type traverseDagNodeState struct {
	// The number of predecessors of this node that we have encountered but not
	// yet finished visiting.
	//
	// INVARIANT: predecessorsOutstanding >= 0
	predecessorsOutstanding int64

	// Have we yet seen this node in the stream of input nodes? If not, we may
	// not yet have encountered all of its predecessors.
	seen bool
}

func (s traverseDagNodeState) checkInvariants() {
	// INVARIANT: predecessorsOutstanding >= 0
	if s.predecessorsOutstanding < 0 {
		log.Panicf("Unexpected count: %d", s.predecessorsOutstanding)
	}
}

func (s traverseDagNodeState) readyToVisit() bool {
	return s.predecessorsOutstanding == 0 && s.seen
}

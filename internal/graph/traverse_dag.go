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
	"fmt"
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
	b := syncutil.NewBundle(ctx)

	// Set up a state struct.
	state := &traverseDAGState{
		notReadyToVisit: make(map[Node]traverseDAGNodeState),
		newReadyNodes:   make(chan Node),
	}

	state.mu = syncutil.NewInvariantMutex(state.checkInvariants)

	// Process incoming nodes from the user.
	b.Add(func(ctx context.Context) (err error) {
		defer close(state.newReadyNodes)
		err = processIncomingNodes(ctx, nodes, sf, state)
		if err != nil {
			err = fmt.Errorf("processIncomingNodes: %v", err)
			return
		}

		return
	})

	// Run multiple workers to deal with the nodes that are ready to visit.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = visitNodes(ctx, v, state)
			if err != nil {
				err = fmt.Errorf("visitNodes: %v", err)
				return
			}

			return
		})
	}

	err = b.Join()
	return
}

type traverseDAGState struct {
	mu syncutil.InvariantMutex

	// A map containing nodes that we are not yet ready to visit, because they
	// have predecessors that have not yet been visited or because we have not
	// yet seen them.
	//
	// INVARIANT: For all v, !v.readyToVisit()
	// INVARIANT: For all v, v.checkInvariants doesn't panic
	//
	// GUARDED_BY(mu)
	notReadyToVisit map[Node]traverseDAGNodeState

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
func (s *traverseDAGState) checkInvariants() {
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

type traverseDAGNodeState struct {
	// The number of predecessors of this node that we have encountered but not
	// yet finished visiting.
	//
	// INVARIANT: predecessorsOutstanding >= 0
	predecessorsOutstanding int64

	// Have we yet seen this node in the stream of input nodes? If not, we may
	// not yet have encountered all of its predecessors.
	seen bool
}

func (s traverseDAGNodeState) checkInvariants() {
	// INVARIANT: predecessorsOutstanding >= 0
	if s.predecessorsOutstanding < 0 {
		log.Panicf("Unexpected count: %d", s.predecessorsOutstanding)
	}
}

func (s traverseDAGNodeState) readyToVisit() bool {
	return s.predecessorsOutstanding == 0 && s.seen
}

// Read incoming nodes and update their successors' records. If the incoming
// node is ready to process, write it to state.newReadyNodes. Otherwise update
// the 'seen' field for its record.
func processIncomingNodes(
	ctx context.Context,
	nodes <-chan Node,
	sf SuccessorFinder,
	state *traverseDAGState) (err error) {
	for n := range nodes {
		// Find the successors for this node.
		var successors []Node
		successors, err = sf.FindDirectSuccessors(ctx, n)
		if err != nil {
			err = fmt.Errorf("FindDirectSuccessors: %v", err)
			return
		}

		state.mu.Lock()

		// Put a hold on each direct successor.
		for _, s := range successors {
			tmp := state.notReadyToVisit[s]
			if tmp.seen {
				log.Panicf("Not topologically sorted? Node: %#v", n)
			}

			tmp.predecessorsOutstanding++
			state.notReadyToVisit[s] = tmp
		}

		// Update state for this node.
		ready := false
		{
			tmp := state.notReadyToVisit[n]
			if tmp.seen {
				log.Panicf("Already seen: %#v", n)
			}

			tmp.seen = true
			if tmp.readyToVisit() {
				ready = true
				delete(state.notReadyToVisit, n)
			} else {
				state.notReadyToVisit[n] = tmp
			}
		}

		state.mu.Unlock()

		// If we're not ready to visit this node, we're done. It's sitting in the
		// map and will be updated later as its predecessors complete.
		if !ready {
			continue
		}

		// We're ready to visit this node. Hand it off to a goroutine performing
		// visits.
		select {
		case state.newReadyNodes <- n:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

// Read nodes that are ready to visit from state.readyToVisit and
// state.newReadyNodes, and visit them. Return when there is no longer any
// possibility of nodes becoming ready.
func visitNodes(
	ctx context.Context,
	v Visitor,
	state *traverseDAGState) (err error) {
	// We enter the loop locked.
	state.mu.Lock()
	for {
		var n Node

		// Grab a node that's immediately available, if any.
		{
			l := len(state.readyToVisit)
			if l > 0 {
				n = state.readyToVisit[l-1]
				state.readyToVisit = state.readyToVisit[:l-1]
			}
		}

		state.mu.Unlock()

		// TODO(jacobsa): Waiting for the channel here isn't great. It means that
		// we may miss updates from another worker when a predecessor finishes, and
		// reduce our parallelism. (The updates won't be lost forever though, since
		// that worker will get around to them.) Hmm. I guess we need to suck it up
		// and use cond vars like in explore.go.
	}
}

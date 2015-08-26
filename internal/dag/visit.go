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

package dag

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/jacobsa/syncutil"

	"golang.org/x/net/context"
)

// Call the visitor once for each unique node in the union of startNodes and
// all of its transitive dependencies, with bounded parallelism.
//
// Guarantees:
//
//  *  If the graph contains a cycle, this function will not succeed.
//
//  *  If a node N depends on a node M, v.Visit(N) will be called only after
//     v.Visit(M) returns successfully.
//
//  *  For each unique node N, dr.FindDependencies(N) and v.Visit(N) will each
//     be called at most once. Moreover, v.Visit(N) will be called only after
//     dr.FindDependencies(N) returns successfully.
//
func Visit(
	ctx context.Context,
	startNodes []Node,
	dr DependencyResolver,
	v Visitor,
	parallelism int) (err error) {
	err = errors.New("TODO")
	return
}

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
//
// The successor finder may be called multiple times per node, and must be
// idempotent.
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

		// One, for processIncomingNodes, to prevent visitNodes from returning
		// immediately. processIncomingNodes will mark itself as no longer busy
		// when it returns.
		busyWorkers: 1,
	}

	state.mu = syncutil.NewInvariantMutex(state.checkInvariants)
	state.cond.L = &state.mu

	// Process incoming nodes from the user.
	b.Add(func(ctx context.Context) (err error) {
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
			err = visitNodes(ctx, sf, v, state)
			if err != nil {
				err = fmt.Errorf("visitNodes: %v", err)
				return
			}

			return
		})
	}

	// Join the bundle, but use the explicitly tracked first worker error in
	// order to circumvent the following race:
	//
	//  *  Worker A encounters an error, sets firstErr, and returns.
	//
	//  *  Worker B wakes up, sees firstErr, and returns with a junk follow-on
	//     error.
	//
	//  *  The bundle observes worker B's error before worker A's.
	//
	b.Join()
	state.mu.Lock()
	err = state.firstErr
	state.mu.Unlock()

	return
}

type visitState struct {
	mu syncutil.InvariantMutex

	// The set of all nodes we have ever admitted to notReadyToVisit or
	// readyToVisit. In other words, the set of all nodes we've yet encountered.
	//
	// GUARDED_BY(mu)
	admitted map[Node]struct{}

	// A map containing nodes that we are not yet ready to visit, because they
	// have dependencies that have not yet been visited or because we have not
	// yet seen them.
	//
	// INVARIANT: For all v, !v.readyToVisit()
	// INVARIANT: For all v, v.checkInvariants doesn't panic
	// INVARIANT: For all v, v is a key in admitted
	//
	// GUARDED_BY(mu)
	notReadyToVisit map[Node]visitNodeState

	// A list of nodes that we are ready to visit but have not yet started
	// visiting.
	//
	// INVARIANT: For all n, n is not a key in notReadyToVisit
	// INVARIANT: For all n, n is a key in admitted
	//
	// GUARDED_BY(mu)
	readyToVisit []Node

	// Set to the first error seen by a worker, if any. When non-nil, all workers
	// should wake up and return.
	//
	// We must track this explicitly rather than just using syncutil.Bundle's
	// support because we sleep on a condition variable, which can't be composed
	// with receiving from the context's Done channel.
	//
	// GUARDED_BY(mu)
	firstErr error

	// The number of workers that are doing something besides waiting on a node
	// to visit. If this hits zero with readyToVisit empty, it means that there
	// is nothing further to do.
	//
	// INVARIANT: busyWorkers >= 0
	//
	// GUARDED_BY(mu)
	busyWorkers int64

	// Broadcasted to with mu held when any of the following state changes:
	//
	//  *  readyToVisit
	//  *  firstErr
	//  *  busyWorkers
	//
	// GUARDED_BY(mu)
	cond sync.Cond
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

	// INVARIANT: busyWorkers >= 0
	if s.busyWorkers < 0 {
		log.Panicf("Negative count: %d", s.busyWorkers)
	}
}

// Is there anything that needs a worker's attention?
//
// LOCKS_REQUIRED(state.mu)
func (state *traverseDAGState) shouldWake() bool {
	return len(state.readyToVisit) != 0 ||
		state.firstErr != nil ||
		state.busyWorkers == 0
}

// Sleep until there's something interesting for a worker to do.
//
// LOCKS_REQUIRED(state.mu)
func (state *traverseDAGState) waitForSomethingToDo() {
	for !state.shouldWake() {
		state.cond.Wait()
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
	// Update state when this function returns.
	defer func() {
		state.mu.Lock()
		defer state.mu.Unlock()

		state.busyWorkers--
		if err != nil && state.firstErr == nil {
			state.firstErr = err
		}

		state.cond.Broadcast()
	}()

	// Deal with each incoming node.
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
		{
			tmp := state.notReadyToVisit[n]
			if tmp.seen {
				log.Panicf("Already seen: %#v", n)
			}

			tmp.seen = true
			if tmp.readyToVisit() {
				delete(state.notReadyToVisit, n)
				state.readyToVisit = append(state.readyToVisit, n)
				state.cond.Broadcast()
			} else {
				state.notReadyToVisit[n] = tmp
			}
		}

		state.mu.Unlock()
	}

	return
}

// Read nodes that are ready to visit from state.readyToVisit and
// state.newReadyNodes, and visit them. Return when there is no longer any
// possibility of nodes becoming ready.
func visitNodes(
	ctx context.Context,
	sf SuccessorFinder,
	v Visitor,
	state *traverseDAGState) (err error) {
	state.mu.Lock()
	defer state.mu.Unlock()

	defer func() {
		// Record our error if it's the first.
		if state.firstErr == nil && err != nil {
			state.firstErr = err
			state.cond.Broadcast()
		}
	}()

	for {
		// Wait for something to do.
		state.waitForSomethingToDo()

		// Why were we awoken?
		switch {
		// Return immediately if another worker has seen an error.
		case state.firstErr != nil:
			err = errors.New("Cancelled")
			return

		// Otherwise, handle work if it exists.
		case len(state.readyToVisit) != 0:
			err = visitOne(ctx, sf, v, state)
			if err != nil {
				return
			}

		// Otherwise, are we done?
		case state.busyWorkers == 0:
			return

		default:
			panic("Unexpected wake-up")
		}
	}
}

// REQUIRES: len(state.readyToVisit) > 0
//
// LOCKS_REQUIRED(state.mu)
func visitOne(
	ctx context.Context,
	sf SuccessorFinder,
	v Visitor,
	state *traverseDAGState) (err error) {
	// Mark this worker as busy for the duration of this function.
	state.busyWorkers++
	state.cond.Broadcast()

	defer func() {
		state.busyWorkers--
		state.cond.Broadcast()
	}()

	// Extract the node to process.
	l := len(state.readyToVisit)
	n := state.readyToVisit[l-1]
	state.readyToVisit = state.readyToVisit[:l-1]
	state.cond.Broadcast()

	// Unlock while visiting and finding successors.
	state.mu.Unlock()
	err = v.Visit(ctx, n)

	var successors []Node
	if err == nil {
		successors, err = sf.FindDirectSuccessors(ctx, n)
	}

	state.mu.Lock()

	// Did we encounter an error in the unlocked region above?
	if err != nil {
		return
	}

	// Update state for the successor nodes. Some may now have been unblocked.
	for _, s := range successors {
		tmp, ok := state.notReadyToVisit[s]
		if !ok {
			log.Panicf("Unexpectedly missing: %#v", s)
		}

		tmp.predecessorsOutstanding--
		if tmp.readyToVisit() {
			delete(state.notReadyToVisit, s)
			state.readyToVisit = append(state.readyToVisit, s)
			state.cond.Broadcast()
		} else {
			state.notReadyToVisit[s] = tmp
		}
	}

	return
}

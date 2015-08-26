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

////////////////////////////////////////////////////////////////////////
// nodeInfo
////////////////////////////////////////////////////////////////////////

type nodeState int

// States in which a node involved in a call to Visit may be.
const (
	// The dependency resolver has not yet been called for this node, or a call
	// is currently in progress.
	state_DependenciesUnresolved nodeState = iota

	// The dependencies of this node have been resolved, but they have not yet
	// all been visited.
	state_DependenciesUnsatisfied

	// This node is eligible to be visited, but the visitor has not yet been
	// called or a call is currently in progress.
	state_Unvisited

	// The visitor has been called for this node and returned successfully.
	state_Visited
)

type nodeInfo struct {
	node  Node
	state nodeState

	// The number of unsatisfied dependencies remaining for this node.
	//
	// INVARIANT: unsatisfied >= 0
	// INVARIANT: unsatisfied > 0 iff state == state_DependenciesUnsatisfied
	unsatisfied int64

	// The set of unsatisfied nodes for which this node is a blocker.
	//
	// INVARIANT: len(dependants) > 0 implies state < state_Unvisited.
	// INVARIANT: For each n, n.state == state_DependenciesUnsatisfied
	dependants []*nodeInfo
}

////////////////////////////////////////////////////////////////////////
// visitState
////////////////////////////////////////////////////////////////////////

type visitState struct {
	mu syncutil.InvariantMutex

	// All of the nodes we've yet encountered and their current state.
	//
	// INVARIANT: For each k, v, v.node == k
	// INVARIANT: For each v, v.checkInvariants() doesn't panic
	//
	// GUARDED_BY(mu)
	nodes map[Node]*nodeInfo

	// The set of nodes in state_DependenciesUnresolved for which we haven't yet
	// started a call to the dependency resolver.
	//
	// INVARIANT: For each v, v.state == state_DependenciesUnresolved
	// INVARIANT: For each v, v.node is a key in nodes
	//
	// GUARDED_BY(mu)
	toResolve []*nodeInfo

	// The set of all nodes in state_DependenciesUnsatisfied. If this is
	// non-empty when we're done, the graph must contain a cycle.
	//
	// INVARIANT: For each k, v.state == state_DependenciesUnsatisfied
	// INVARIANT: For each k, v.node is a key in nodes
	//
	// GUARDED_BY(mu)
	unsatisfied map[*nodeInfo]struct{}

	// The set of nodes in state_Unvisited for which we haven't yet started a
	// call to the visitor.
	//
	// INVARIANT: For each v, v.state == state_Unvisited
	// INVARIANT: For each v, v.node is a key in nodes
	//
	// GUARDED_BY(mu)
	toVisit []*nodeInfo

	// Set to the first error seen by a worker, if any. When non-nil, all workers
	// should wake up and return.
	//
	// We must track this explicitly rather than just using syncutil.Bundle's
	// support because we sleep on a condition variable, which can't be composed
	// with receiving from the context's Done channel.
	//
	// GUARDED_BY(mu)
	firstErr error

	// The number of workers that are doing something besides waiting on work. If
	// this hits zero with toResolve and toVisit both empty, it means that there
	// is nothing further to do.
	//
	// INVARIANT: busyWorkers >= 0
	//
	// GUARDED_BY(mu)
	busyWorkers int64

	// Broadcasted to with mu held when any of the following state changes:
	//
	//  *  toResolve
	//  *  toVisit
	//  *  firstErr
	//  *  busyWorkers
	//
	// See also visitState.shouldWake, with which this list must be kept in sync.
	//
	// GUARDED_BY(mu)
	cond sync.Cond
}

// LOCKS_REQUIRED(s.mu)
func (s *visitState) checkInvariants() {
	panic("TODO")
}

// Is there anything that needs a worker's attention?
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) shouldWake() bool {
	return len(state.toResolve) != 0 ||
		len(state.toVisit) != 0 ||
		state.firstErr != nil ||
		state.busyWorkers == 0
}

// Sleep until there's something interesting for a worker to do.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) waitForSomethingToDo() {
	for !state.shouldWake() {
		state.cond.Wait()
	}
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

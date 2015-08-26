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
	b := syncutil.NewBundle(ctx)

	// Set up a state struct.
	state := &visitState{
		dr:          dr,
		visitor:     v,
		nodes:       make(map[Node]*nodeInfo),
		unsatisfied: make(map[*nodeInfo]struct{}),
	}

	state.mu = syncutil.NewInvariantMutex(state.checkInvariants)
	state.cond.L = &state.mu

	// Add each of the start nodes.
	state.mu.Lock()
	state.addNodes(startNodes)
	state.mu.Unlock()

	// Run multiple workers.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = state.processNodes(ctx)
			if err != nil {
				err = fmt.Errorf("processNodes: %v", err)
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
	//  *  Worker B wakes up, sees firstErr, and returns with a follow-on
	//     "cancelled" error.
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
	// INVARIANT: For each v, v.state == state_DependenciesUnsatisfied
	dependants []*nodeInfo
}

func (ni *nodeInfo) checkInvariants() {
	// INVARIANT: unsatisfied >= 0
	if !(ni.unsatisfied >= 0) {
		log.Panicf("unsatisfied: %d", ni.unsatisfied)
	}

	// INVARIANT: unsatisfied > 0 iff state == state_DependenciesUnsatisfied
	if (ni.unsatisfied > 0) != (ni.state == state_DependenciesUnsatisfied) {
		log.Panicf("unsatisfied: %d, state: %v", ni.unsatisfied, ni.state)
	}

	// INVARIANT: len(dependants) > 0 implies state < state_Unvisited.
	if len(ni.dependants) > 0 && !(ni.state < state_Unvisited) {
		log.Panicf("dependants: %d, state: %v", len(ni.dependants), ni.state)
	}

	// INVARIANT: For each v, v.state == state_DependenciesUnsatisfied
	for _, dep := range ni.dependants {
		if dep.state != state_DependenciesUnsatisfied {
			log.Panicf("dep.state: %v", dep.state)
		}
	}
}

////////////////////////////////////////////////////////////////////////
// visitState
////////////////////////////////////////////////////////////////////////

type visitState struct {
	dr      DependencyResolver
	visitor Visitor

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
	// INVARIANT: For each k, k.state == state_DependenciesUnsatisfied
	// INVARIANT: For each k, k.node is a key in nodes
	// INVARIANT: All unsatisfied elements of nodes are in unsatisfied
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
	// INVARIANT: For each k, v, v.node == k
	for k, v := range s.nodes {
		if !(v.node == k) {
			log.Panicf("Node mismatch: %#v, %#v", v.node, k)
		}
	}

	// INVARIANT: For each v, v.checkInvariants() doesn't panic
	for _, v := range s.nodes {
		v.checkInvariants()
	}

	// INVARIANT: For each v, v.state == state_DependenciesUnresolved
	for _, v := range s.toResolve {
		if !(v.state == state_DependenciesUnresolved) {
			log.Panicf("Unexpected state: %v", v.state)
		}
	}

	// INVARIANT: For each v, v.node is a key in nodes
	for _, v := range s.toResolve {
		_, ok := s.nodes[v.node]
		if !ok {
			log.Panicf("Unknown node: %#v", v)
		}
	}

	// INVARIANT: For each k, k.state == state_DependenciesUnsatisfied
	for k, _ := range s.unsatisfied {
		if !(k.state == state_DependenciesUnsatisfied) {
			log.Panicf("Unexpected state: %v", k.state)
		}
	}

	// INVARIANT: For each k, k.node is a key in nodes
	for k, _ := range s.unsatisfied {
		_, ok := s.nodes[k.node]
		if !ok {
			log.Panicf("Unknown node: %#v", k)
		}
	}

	// INVARIANT: All unsatisfied elements of nodes are in unsatisfied
	for _, ni := range s.nodes {
		if ni.state != state_DependenciesUnsatisfied {
			continue
		}

		_, ok := s.unsatisfied[ni]
		if !ok {
			log.Panicf("Missing unsatisfied node: %#v", ni.node)
		}
	}

	// INVARIANT: For each v, v.state == state_Unvisited
	for _, v := range s.toVisit {
		if !(v.state == state_Unvisited) {
			log.Panicf("Unexpected state: %v", v.state)
		}
	}

	// INVARIANT: For each v, v.node is a key in nodes
	for _, v := range s.toVisit {
		_, ok := s.nodes[v]
		if !ok {
			log.Panicf("Unknown node: %#v", v)
		}
	}

	// INVARIANT: busyWorkers >= 0
	if !(s.busyWorkers >= 0) {
		log.Panicf("busyWorkers: %d", s.busyWorkers)
	}
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

// Add any nodes from the list that we haven't yet seen as new nodeInfo structs
// in state_DependenciesUnresolved. Ignore those that we have seen.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) addNodes(nodes []Node) {
	for _, n := range nodes {
		// Skip nodes that we already know.
		if _, ok := state.nodes[n]; ok {
			continue
		}

		ni := &nodeInfo{
			node:  n,
			state: state_DependenciesUnresolved,
		}

		state.nodes[n] = ni
		state.toResolve = append(state.toResolve, ni)
	}
}

// Given a node that was removed from toResolve, unsatisfied, or toVisit and
// then updated, re-insert it in the appropriate place.
func (state *visitState) reinsert(ni *nodeInfo) {
	panic("TODO")
}

// Watch for nodes that can be resolved or visited and do so. Return when it's
// guaranteed that there's nothing further to do.
func (state *visitState) processNodes(ctx context.Context) (err error) {
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

		// Is there a node that can be visited?
		case len(state.toVisit) > 0:
			err = state.visitOne(ctx)
			if err != nil {
				err = fmt.Errorf("visitOne: %v", err)
				return
			}

		// Is there a node that needs to be resolved?
		case len(state.toResolve) > 0:
			err = state.resolveOne(ctx)
			if err != nil {
				err = fmt.Errorf("resolveOne: %v", err)
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

// REQUIRES: len(state.toVisit) > 0
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) visitOne(ctx context.Context) (err error) {
	// Mark this worker as busy for the duration of this function.
	state.busyWorkers++
	state.cond.Broadcast()

	defer func() {
		state.busyWorkers--
		state.cond.Broadcast()
	}()

	// Extract the node to visit.
	l := len(state.toVisit)
	ni := state.toVisit[l-1]
	state.toVisit = state.toVisit[:l-1]
	state.cond.Broadcast()

	// Unlock while visiting.
	state.mu.Unlock()
	err = state.visitor.Visit(ctx, ni.node)
	state.mu.Lock()

	// Did we encounter an error in the unlocked region above?
	if err != nil {
		return
	}

	// Update each dependant, now that this node has been visited.
	for _, dep := range ni.dependants {
		dep.unsatisfied--
		if dep.unsatisfied == 0 {
			delete(state.unsatisfied, dep)
			state.reinsert(dep)
		}
	}

	// Update the node itself.
	ni.state = state_Visited

	return
}

// REQUIRES: len(state.toResolve) > 0
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) resolveOne(ctx context.Context) (err error) {
	err = errors.New("TODO: Model this on visitOne.")
	return
}

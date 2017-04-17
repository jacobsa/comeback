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
	"context"
	"fmt"
	"log"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/syncutil"
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
	resolverParallelism int,
	visitorParallelism int) (err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// Set up a state struct.
	state := &visitState{
		dr:          dr,
		visitor:     v,
		nodes:       make(map[Node]*nodeInfo),
		unsatisfied: make(map[*nodeInfo]struct{}),
	}

	state.mu = syncutil.NewInvariantMutex(state.checkInvariants)
	state.wakeResolvers.L = &state.mu
	state.wakeVisitors.L = &state.mu

	// Add each of the start nodes.
	state.mu.Lock()
	state.addNodes(startNodes)
	state.mu.Unlock()

	// Run workers.
	for i := 0; i < resolverParallelism; i++ {
		eg.Go(func() (err error) {
			err = state.resolveNodes(ctx)
			if err != nil {
				err = fmt.Errorf("resolveNodes: %v", err)
				return
			}

			return
		})
	}

	for i := 0; i < visitorParallelism; i++ {
		eg.Go(func() (err error) {
			err = state.visitNodes(ctx)
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
	//  *  Worker B wakes up, sees firstErr, and returns with a follow-on
	//     "cancelled" error.
	//
	//  *  The bundle observes worker B's error before worker A's.
	//
	eg.Wait()
	state.mu.Lock()
	err = state.firstErr
	state.mu.Unlock()

	if err != nil {
		return
	}

	// If we ran out of work to do while some nodes still had unsatisfied
	// dependencies, the graph has a cycle.
	if len(state.unsatisfied) > 0 {
		var someNode Node
		for ni, _ := range state.unsatisfied {
			someNode = ni.node
			break
		}

		err = fmt.Errorf(
			"Graph contains a cycle causing unsatisfied node: %#v",
			someNode)

		return
	}

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
	// INVARIANT: depsUnsatisfied >= 0
	// INVARIANT: depsUnsatisfied > 0 iff state == state_DependenciesUnsatisfied
	depsUnsatisfied int64

	// The set of unsatisfied nodes for which this node is a blocker.
	//
	// INVARIANT: len(dependants) > 0 implies state < state_Visited.
	// INVARIANT: For each v, v.state == state_DependenciesUnsatisfied
	dependants []*nodeInfo
}

func (ni *nodeInfo) checkInvariants() {
	// INVARIANT: depsUnsatisfied >= 0
	if !(ni.depsUnsatisfied >= 0) {
		log.Panicf("depsUnsatisfied: %d", ni.depsUnsatisfied)
	}

	// INVARIANT: depsUnsatisfied > 0 iff state == state_DependenciesUnsatisfied
	if (ni.depsUnsatisfied > 0) != (ni.state == state_DependenciesUnsatisfied) {
		log.Panicf("depsUnsatisfied: %d, state: %v", ni.depsUnsatisfied, ni.state)
	}

	// INVARIANT: len(dependants) > 0 implies state < state_Visited.
	if len(ni.dependants) > 0 && !(ni.state < state_Visited) {
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

	// The set of all nodes in state_DependenciesUnsatisfied. The graph contains
	// a cycle iff this is non-empty when we run out of work to do.
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

	// The number of resolve workers that are doing something besides waiting on
	// work.
	//
	// INVARIANT: busyResolvers >= 0
	//
	// GUARDED_BY(mu)
	busyResolvers int64

	// The number of visitor workers that are doing something besides waiting on
	// work.
	//
	// INVARIANT: busyVisitors >= 0
	//
	// GUARDED_BY(mu)
	busyVisitors int64

	// Broadcasted to with mu held when there may be an action that can be taken
	// by currently-sleeping resolvers.
	//
	// GUARDED_BY(mu)
	wakeResolvers sync.Cond

	// Broadcasted to with mu held when there may be an action that can be taken
	// by currently-sleeping visitors.
	//
	// GUARDED_BY(mu)
	wakeVisitors sync.Cond
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
		_, ok := s.nodes[v.node]
		if !ok {
			log.Panicf("Unknown node: %#v", v)
		}
	}

	// INVARIANT: busyResolvers >= 0
	if !(s.busyResolvers >= 0) {
		log.Panicf("busyResolvers: %d", s.busyResolvers)
	}

	// INVARIANT: busyVisitors >= 0
	if !(s.busyVisitors >= 0) {
		log.Panicf("busyVisitors: %d", s.busyVisitors)
	}
}

// Run the supplied function, which updates the state struct in some way under
// the state's lock. If the update causes a wakeup event, signal the
// appropriate condition variable.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) update(f func()) {
	resolversAwakeBefore := state.resolversShouldWake()
	visitorsAwakeBefore := state.visitorsShouldWake()

	f()

	if !resolversAwakeBefore && state.resolversShouldWake() {
		state.wakeResolvers.Broadcast()
	}

	if !visitorsAwakeBefore && state.visitorsShouldWake() {
		state.wakeVisitors.Broadcast()
	}
}

// Run the supplied function, which may fail. If it fails and is the first such
// action to fail, update state.firstErr using state.update.
//
// The lock will be held on entry to the action, and must be held on exit from
// the action.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) runAction(f func() error) (err error) {
	err = f()
	if err != nil && state.firstErr == nil {
		state.update(func() { state.firstErr = err })
	}

	return
}

// Should the resolver workers stop processing work?
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) resolversShouldExit() bool {
	// If some other worker has seen an error, we stop immediately.
	if state.firstErr != nil {
		return true
	}

	// If there is something for us to do, we should not exit.
	if len(state.toResolve) != 0 {
		return false
	}

	// If there is a worker that may soon provide something further to resolve,
	// we must not stop.
	if state.busyResolvers != 0 {
		return false
	}

	// At this point we know there can be nothing further for us to do.
	return true
}

// Should the visitor workers stop processing work?
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) visitorsShouldExit() bool {
	// If some other worker has seen an error, we stop immediately.
	if state.firstErr != nil {
		return true
	}

	// If there is something for us to do, we should not exit.
	if len(state.toVisit) != 0 {
		return false
	}

	// If there is some visit in process, it may cause an unsatisfied node to
	// become satisfied, so we must wait.
	if state.busyVisitors != 0 {
		return false
	}

	// If the dependency resolvers may yet provide more work for us, we must
	// wait.
	if !state.resolversShouldExit() {
		return false
	}

	// At this point we know there can be nothing further for us to do.
	return true
}

// Is there anything that needs a resolver worker's attention?
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) resolversShouldWake() bool {
	return len(state.toResolve) != 0 || state.resolversShouldExit()
}

// Is there anything that needs a visitor worker's attention?
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) visitorsShouldWake() bool {
	return len(state.toVisit) != 0 || state.visitorsShouldExit()
}

// Sleep until there's something interesting for a resolver worker to do.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) waitForResolverWork() {
	for !state.resolversShouldWake() {
		state.wakeResolvers.Wait()
	}
}

// Sleep until there's something interesting for a visitor worker to do.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) waitForVisitorWork() {
	for !state.visitorsShouldWake() {
		state.wakeVisitors.Wait()
	}
}

// Add any nodes from the list that we haven't yet seen as new nodeInfo structs
// in state_DependenciesUnresolved. Ignore those that we have seen.
//
// Must be run under state.update.
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
		state.reinsert(ni)
	}
}

// Given a node that was removed from toResolve, unsatisfied, or toVisit and
// then updated, re-insert it in the appropriate place.
//
// Must be run under state.update.
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) reinsert(ni *nodeInfo) {
	switch ni.state {
	default:
		log.Panicf("Unknown state: %#v", ni)

	case state_DependenciesUnresolved:
		state.toResolve = append(state.toResolve, ni)

	case state_DependenciesUnsatisfied:
		state.unsatisfied[ni] = struct{}{}

	case state_Unvisited:
		state.toVisit = append(state.toVisit, ni)

	case state_Visited:
		// Nothing to do.
	}
}

// Watch for nodes that can be resolved and do so. Return when it's guaranteed
// that there's nothing further to do.
//
// LOCKS_EXCLUDED(state.mu)
func (state *visitState) resolveNodes(ctx context.Context) error {
	state.mu.Lock()
	defer state.mu.Unlock()

	return state.runAction(func() (err error) {
		for {
			// Wait for something to do.
			state.waitForResolverWork()

			// Why were we awoken?
			switch {
			// Should we exit?
			case state.resolversShouldExit():
				return

			// Is there a node that needs to be resolved?
			case len(state.toResolve) > 0:
				err = state.resolveOne(ctx)
				if err != nil {
					err = fmt.Errorf("resolveOne: %v", err)
					return
				}

			default:
				panic("Unexpected wake-up")
			}
		}
	})
}

// Watch for nodes that can be visited and do so. Return when it's guaranteed
// that there's nothing further to do.
func (state *visitState) visitNodes(ctx context.Context) error {
	state.mu.Lock()
	defer state.mu.Unlock()

	return state.runAction(func() (err error) {
		for {
			// Wait for something to do.
			state.waitForVisitorWork()

			// Why were we awoken?
			switch {
			// Should we exit?
			case state.visitorsShouldExit():
				return

			// Is there a node that can be visited?
			case len(state.toVisit) > 0:
				err = state.visitOne(ctx)
				if err != nil {
					err = fmt.Errorf("visitOne: %v", err)
					return
				}

			default:
				panic("Unexpected wake-up")
			}
		}
	})
}

// REQUIRES: len(state.toVisit) > 0
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) visitOne(ctx context.Context) (err error) {
	// Mark this worker as busy for the duration of this function.
	state.busyVisitors++
	defer state.update(func() { state.busyVisitors-- })

	// Extract the node to visit.
	var ni *nodeInfo
	state.update(func() {
		l := len(state.toVisit)
		ni = state.toVisit[l-1]
		state.toVisit = state.toVisit[:l-1]
	})

	// Unlock while visiting.
	state.mu.Unlock()
	err = state.visitor.Visit(ctx, ni.node)
	state.mu.Lock()

	// Did we encounter an error in the unlocked region above?
	if err != nil {
		return
	}

	// Perform state updates.
	state.update(func() {
		// Update each dependant, now that this node has been visited.
		for _, dep := range ni.dependants {
			dep.depsUnsatisfied--
			if dep.depsUnsatisfied == 0 {
				dep.state = state_Unvisited
				delete(state.unsatisfied, dep)
				state.reinsert(dep)
			}
		}

		// Update and reinsert the node itself.
		ni.dependants = nil
		ni.state = state_Visited
		state.reinsert(ni)
	})

	return
}

// REQUIRES: len(state.toResolve) > 0
//
// LOCKS_REQUIRED(state.mu)
func (state *visitState) resolveOne(ctx context.Context) (err error) {
	// Mark this worker as busy for the duration of this function.
	state.busyResolvers++
	defer state.update(func() { state.busyResolvers-- })

	// Extract the node to resolve.
	var ni *nodeInfo
	state.update(func() {
		l := len(state.toResolve)
		ni = state.toResolve[l-1]
		state.toResolve = state.toResolve[:l-1]
	})

	// Unlock while resolving.
	state.mu.Unlock()
	deps, err := state.dr.FindDependencies(ctx, ni.node)
	state.mu.Lock()

	// Did we encounter an error in the unlocked region above?
	if err != nil {
		return
	}

	// Perform state updates.
	state.update(func() {
		// Ensure that we have a record for every dependency.
		state.addNodes(deps)

		// Add this node to the list of dependants for each dependency that hasn't
		// yet been visited. Count them as we go.
		for _, dep := range deps {
			depNi := state.nodes[dep]
			if depNi.state == state_Visited {
				continue
			}

			ni.depsUnsatisfied++
			depNi.dependants = append(depNi.dependants, ni)
		}

		// Update and reinsert the node itself.
		if ni.depsUnsatisfied > 0 {
			ni.state = state_DependenciesUnsatisfied
		} else {
			ni.state = state_Unvisited
		}

		state.reinsert(ni)
	})

	return
}

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
	"fmt"
	"sync"

	"github.com/jacobsa/syncutil"

	"golang.org/x/net/context"
)

// Given a set S of root nodes within the directed graph defined by the
// supplied successor finder, write all nodes accessible from the nodes in S to
// the supplied channel, without duplicates. There is no guarantee on output
// order.
//
// The successor finder may be called up to the given number of times
// concurrently.
func ExploreDirectedGraph(
	ctx context.Context,
	sf SuccessorFinder,
	roots []Node,
	nodes chan<- Node,
	parallelism int) (err error) {
	b := syncutil.NewBundle(ctx)

	// Set up initial state.
	es := &exploreState{
		output:   nodes,
		admitted: make(map[Node]struct{}),
	}

	es.mu = syncutil.NewInvariantMutex(es.checkInvariants)
	es.cond.L = &es.mu

	es.mu.Lock()
	es.enqueueNodes(roots)
	es.mu.Unlock()

	// Run the appropriate number of workers.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = explore(ctx, es, sf)
			return
		})
	}

	// Join the bundle, but use the explicitly tracked first worker error in
	// order to circumvent the following race:
	//
	//  *  Worker A encounters an error, sets firstErr, and returns
	//
	//  *  Worker B wakes up, sees firstErr, and returns with a junk follow-on
	//     error.
	//
	//  *  The bundle observes worker B's error before worker A's.
	//
	b.Join()
	es.mu.Lock()
	err = es.firstErr
	es.mu.Unlock()

	return
}

////////////////////////////////////////////////////////////////////////
// exploreState
////////////////////////////////////////////////////////////////////////

// State shared by each ExploreDirectedGraph worker.
type exploreState struct {
	output chan<- Node
	mu     syncutil.InvariantMutex

	// All nodes that have ever been seen. If a node is in this map, it will
	// eventually be visted (barring errors returned by the visitor).
	//
	// GUARDED_BY(mu)
	admitted map[Node]struct{}

	// Admitted nodes that have yet to be visted.
	//
	// INVARIANT: For each n in toVisit, n is a key of admitted.
	//
	// GUARDED_BY(mu)
	toVisit []Node

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
	// to visit. If this hits zero with toVisit empty, it means that there is
	// nothing further to do.
	//
	// GUARDED_BY(mu)
	busyWorkers int

	// Broadcasted to with mu held when any of the following state changes:
	//
	//  *  toVisit
	//  *  firstErr
	//  *  busyWorkers
	//
	// GUARDED_BY(mu)
	cond sync.Cond
}

// LOCKS_REQUIRED(es.mu)
func (es *exploreState) checkInvariants() {
	// INVARIANT: For each n in toVisit, n is a key of admitted.
	for _, n := range es.toVisit {
		if _, ok := es.admitted[n]; !ok {
			panic(fmt.Sprintf("Expected %q to be in admitted map", n))
		}
	}
}

// Is there anything that needs a worker's attention?
//
// LOCKS_REQUIRED(es.mu)
func (es *exploreState) shouldWake() bool {
	return len(es.toVisit) != 0 || es.firstErr != nil || es.busyWorkers == 0
}

// Sleep until there's something interesting for a worker to do.
//
// LOCKS_REQUIRED(es.mu)
func (es *exploreState) waitForSomethingToDo() {
	for !es.shouldWake() {
		es.cond.Wait()
	}
}

// Enqueue any of the supplied nodes that haven't already been enqueued.
//
// LOCKS_REQUIRED(es.mu)
func (es *exploreState) enqueueNodes(nodes []Node) {
	for _, n := range nodes {
		if _, ok := es.admitted[n]; !ok {
			es.toVisit = append(es.toVisit, n)
			es.admitted[n] = struct{}{}
		}
	}

	es.cond.Broadcast()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// REQUIRES: len(es.toVisit) > 0
//
// LOCKS_REQUIRED(es.mu)
func exploreOne(
	ctx context.Context,
	es *exploreState,
	sf SuccessorFinder) (err error) {
	// Mark this worker as busy for the duration of this function.
	es.busyWorkers++
	es.cond.Broadcast()

	defer func() {
		es.busyWorkers--
		es.cond.Broadcast()
	}()

	// Extract the node to process.
	l := len(es.toVisit)
	node := es.toVisit[l-1]
	es.toVisit = es.toVisit[:l-1]
	es.cond.Broadcast()

	// Unlock while writing to the channel and visiting.
	es.mu.Unlock()

	var successors []Node
	select {
	case es.output <- node:
		successors, err = sf.FindDirectSuccessors(ctx, node)

	case <-ctx.Done():
		err = ctx.Err()
	}

	es.mu.Lock()

	// Did we encounter an error in the unlocked region above?
	if err != nil {
		return
	}

	// Enqueue the successor nodes that we haven't already admitted.
	es.enqueueNodes(successors)

	return
}

// A single ExploreDirectedGraph worker.
func explore(
	ctx context.Context,
	es *exploreState,
	sf SuccessorFinder) (err error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	defer func() {
		// Record our error if it's the first.
		if es.firstErr == nil && err != nil {
			es.firstErr = err
			es.cond.Broadcast()
		}
	}()

	for {
		// Wait for something to do.
		es.waitForSomethingToDo()

		// Why were we awoken?
		switch {
		// Return immediately if another worker has seen an error.
		case es.firstErr != nil:
			err = errors.New("Cancelled")
			return

		// Otherwise, handle work if it exists.
		case len(es.toVisit) != 0:
			err = exploreOne(ctx, es, sf)
			if err != nil {
				return
			}

		// Otherwise, are we done?
		case es.busyWorkers == 0:
			return

		default:
			panic("Unexpected wake-up")
		}
	}
}

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

	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

// A visitor in a directed graph whose nodes are identified by strings.
type Visitor interface {
	// Process the supplied node and return a list of direct successors.
	Visit(ctx context.Context, node string) (adjacent []string, err error)
}

// Invoke v.Visit once on each node reachable from the supplied search roots,
// including the roots themselves. Use the supplied degree of parallelism.
//
// It is guaranteed that if a node N is fed to v.Visit, then either:
//
//  *  N is an element of roots, or
//  *  There exists a direct predecessor N' of N such that v.Visit(N') was
//     called and returned successfully.
//
// In particular, if the graph is a rooted tree and searching starts at the
// root, then parents will be successfully visited before children are visited.
// However note that in arbitrary DAGs it is *not* guaranteed that all of a
// node's predecessors have been visited before it is.
func Traverse(
	ctx context.Context,
	parallelism int,
	roots []string,
	v Visitor) (err error) {
	b := syncutil.NewBundle(ctx)

	// Set up initial state.
	ts := &traverseState{
		admitted: make(map[string]struct{}),
	}

	ts.mu = syncutil.NewInvariantMutex(ts.checkInvariants)
	ts.cond.L = &ts.mu

	ts.mu.Lock()
	ts.enqueueNodes(roots)
	ts.mu.Unlock()

	// Run the appropriate number of workers.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = traverse(ctx, ts, v)
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
	ts.mu.Lock()
	err = ts.firstErr
	ts.mu.Unlock()

	return
}

////////////////////////////////////////////////////////////////////////
// traverseState
////////////////////////////////////////////////////////////////////////

// State shared by each traverse worker.
type traverseState struct {
	mu syncutil.InvariantMutex

	// All nodes that have ever been seen. If a node is in this map, it will
	// eventually be visted (barring errors returned by the visitor).
	//
	// GUARDED_BY(mu)
	admitted map[string]struct{}

	// Admitted nodes that have yet to be visted.
	//
	// INVARIANT: For each n in toVisit, n is a key of admitted.
	//
	// GUARDED_BY(mu)
	toVisit []string

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

// LOCKS_REQUIRED(ts.mu)
func (ts *traverseState) checkInvariants() {
	// INVARIANT: For each n in toVisit, n is a key of admitted.
	for _, n := range ts.toVisit {
		if _, ok := ts.admitted[n]; !ok {
			panic(fmt.Sprintf("Expected %q to be in admitted map", n))
		}
	}
}

// Is there anything that needs a worker's attention?
//
// LOCKS_REQUIRED(ts.mu)
func (ts *traverseState) shouldWake() bool {
	return len(ts.toVisit) != 0 || ts.firstErr != nil || ts.busyWorkers == 0
}

// Sleep until there's something interesting for a traverse worker.
//
// LOCKS_REQUIRED(ts.mu)
func (ts *traverseState) waitForSomethingToDo() {
	for !ts.shouldWake() {
		ts.cond.Wait()
	}
}

// Enqueue any of the supplied nodes that haven't already been enqueued.
//
// LOCKS_REQUIRED(ts.mu)
func (ts *traverseState) enqueueNodes(nodes []string) {
	for _, n := range nodes {
		if _, ok := ts.admitted[n]; !ok {
			ts.toVisit = append(ts.toVisit, n)
			ts.admitted[n] = struct{}{}
		}
	}

	ts.cond.Broadcast()
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// REQUIRES: len(ts.toVisit) > 0
//
// LOCKS_REQUIRED(ts.mu)
func visitOne(
	ctx context.Context,
	ts *traverseState,
	v Visitor) (err error) {
	// Mark this worker as busy for the duration of this function.
	ts.busyWorkers++
	ts.cond.Broadcast()

	defer func() {
		ts.busyWorkers--
		ts.cond.Broadcast()
	}()

	// Extract the node to visit.
	l := len(ts.toVisit)
	node := ts.toVisit[l-1]
	ts.toVisit = ts.toVisit[:l-1]
	ts.cond.Broadcast()

	// Unlock while visiting.
	ts.mu.Unlock()
	adjacent, err := v.Visit(ctx, node)
	ts.mu.Lock()

	if err != nil {
		return
	}

	// Enqueue the adjacent nodes that we haven't already admitted.
	ts.enqueueNodes(adjacent)

	return
}

// A single traverse worker.
func traverse(
	ctx context.Context,
	ts *traverseState,
	v Visitor) (err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	defer func() {
		// Record our error if it's the first.
		if ts.firstErr == nil && err != nil {
			ts.firstErr = err
			ts.cond.Broadcast()
		}
	}()

	for {
		// Wait for something to do.
		ts.waitForSomethingToDo()

		// Why were we awoken?
		switch {
		// Return immediately if another worker has seen an error.
		case ts.firstErr != nil:
			err = errors.New("Cancelled")
			return

		// Otherwise, handle work if it exists.
		case len(ts.toVisit) != 0:
			err = visitOne(ctx, ts, v)
			if err != nil {
				return
			}

		// Otherwise, are we done?
		case ts.busyWorkers == 0:
			return

		default:
			panic("Unexpected wake-up")
		}
	}
}

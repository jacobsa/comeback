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

	// Ensure that ts.cancelled is set when the context is cancelled (or when we
	// return from this function, if the context will never be cancelled).
	done := ctx.Done()
	if done == nil {
		doneChan := make(chan struct{})
		defer close(doneChan)

		done = doneChan
	}

	go watchForCancel(done, ts)

	// Run the appropriate number of workers.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = traverse(ctx, ts, v)
			return
		})
	}

	err = b.Join()
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

	// Set to true if the context has been cancelled. All workers should return
	// when this happens.
	//
	// GUARDED_BY(mu)
	cancelled bool

	// The number of workers that are doing something besides waiting on a node
	// to visit. If this hits zero with toVisit empty, it means that there is
	// nothing further to do.
	//
	// GUARDED_BY(mu)
	busyWorkers int

	// Broadcasted to with mu held when any of the following state changes:
	//
	//  *  toVisit
	//  *  cancelled
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
	return len(ts.toVisit) != 0 || ts.cancelled || ts.busyWorkers == 0
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

	for {
		// Wait for something to do.
		ts.waitForSomethingToDo()

		// Why were we awoken?
		switch {
		// Return immediately if cancelled.
		case ts.cancelled:
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

// Bridge context cancellation with traverseState.cancelled.
func watchForCancel(
	done <-chan struct{},
	ts *traverseState) {
	<-done

	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.cancelled = true
	ts.cond.Broadcast()
}

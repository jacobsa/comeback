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
	"sync"

	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

// A visitor in a directed graph whose nodes are identified by strings.
type Visitor interface {
	// Process the supplied node and return a list of direct successors.
	Visit(ctx context.Context, node string) (adjacent []string, err error)
}

// Invoke v.Visit on each node reachable from the supplied search roots,
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
	defer func() { err = b.Join() }()

	// Set up initial state.
	ts := &traverseState{
		admitted: make(map[string]struct{}),
	}

	ts.mu = syncutil.NewInvariantMutex(ts.checkInvariants)
	ts.cond.L = &ts.mu

	for _, r := range roots {
		ts.admitted[r] = struct{}{}
		ts.toVisit = append(ts.toVisit, r)
	}

	// Ensure that ts.cancelled is set when the context is eventually cancelled.
	go watchForCancel(ctx, ts)

	// Run the appropriate number of workers.
	for i := 0; i < parallelism; i++ {
		b.Add(func(ctx context.Context) (err error) {
			err = traverse(ctx, ts, v)
			return
		})
	}

	return
}

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

	// Signalled with mu held when any of the following state changes:
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
	panic("TODO")
}

// A single traverse worker.
func traverse(
	ctx context.Context,
	ts *traverseState,
	v Visitor) (err error) {
	err = errors.New("TODO")
	return
}

// Bridge context cancellation with traverseState.cancelled.
func watchForCancel(
	ctx context.Context,
	ts *traverseState) {
	panic("TODO")
}

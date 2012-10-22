// Copyright 2012 Aaron Jacobs. All Rights Reserved.
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

package concurrent

import (
	"runtime"
)

// An item of work, used by the Executor interface.
type Work func()

// An executor accepts work to be run, and runs it at some point in the future.
// It may have a fixed-length buffer of work.
type Executor interface {
	// Add work to the queue. This function may block while other work is in
	// progress. For this reason, the work should not itself call Add
	// synchronously.
	//
	// There are no guarantees on the order in which scheduled work is run.
	Add(w Work)
}

// Create an executor with the specified number of workers running in parallel.
// Calls to Add will block if numWorkers pieces of work are currently in
// progress. numWorkers must be non-zero.
func NewExecutor(numWorkers int) Executor {
	if numWorkers == 0 {
		panic("numWorkers must be non-zero.")
	}

	e := &executor{}
	startWorkers(e, numWorkers)
	runtime.SetFinalizer(e, stopWorkers)

	return e
}

type executor struct {
	workChan chan<- Work
}

func startWorkers(e *executor, numWorkers int) {
	workChan := make(chan Work)
	e.workChan = workChan

	processWork := func() {
		for w := range workChan {
			w()
		}
	}

	for i := 0; i < numWorkers; i++ {
		go processWork()
	}
}

func stopWorkers(e *executor) {
	close(e.workChan)
}

func (e *executor) Add(w Work) {
	e.workChan <- w
}

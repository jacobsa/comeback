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

package concurrent_test

import (
	"github.com/jacobsa/comeback/concurrent"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
	"time"
)

func TestExecutor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ExecutorTest struct {
}

func init() { RegisterTestSuite(&ExecutorTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ExecutorTest) NumWorkersZero() {
	f := func() { concurrent.NewExecutor(0) }
	ExpectThat(f, Panics(HasSubstr("non-zero")))
}

func (t *ExecutorTest) NumWorkersOne() {
	e := concurrent.NewExecutor(1)

	// Set up a piece of work that blocks for awhile then returns.
	sleepDuration := 100 * time.Millisecond
	w := concurrent.Work(func() { time.Sleep(sleepDuration) })

	// Schedule it several times and record the amount of time it takes each
	// instance.
	waitTimes := []time.Duration{}
	const numIters = 16;
	for i := 0; i < numIters; i++ {
		before := time.Now()
		e.Add(w)
		after := time.Now()

		waitTimes = append(waitTimes, after.Sub(before))
	}

	// The first call should have been quick.
	ExpectLt(waitTimes[0], time.Duration(float64(sleepDuration) * 0.10))

	// All of the others should have taken about as long as the piece of work
	// sleeps.
	for i := 1; i < len(waitTimes); i++ {
		ExpectGt(waitTimes[i], time.Duration(float64(sleepDuration) * 0.75))
		ExpectLt(waitTimes[i], time.Duration(float64(sleepDuration) * 1.25))
	}
}

func (t *ExecutorTest) NumWorkersSixteen() {
	ExpectEq("TODO", "")
}

func (t *ExecutorTest) ShutsDownWorkers() {
	ExpectEq("TODO", "")
}

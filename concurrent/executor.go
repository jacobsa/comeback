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

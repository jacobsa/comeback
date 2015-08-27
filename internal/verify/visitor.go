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

package verify

import (
	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/timeutil"
)

// Create a visitor that confirms file chunks can be loaded from the supplied
// blob store, writing out appropriate records to certify this. Nothing is done
// for directory nodes (which are handled by the dependency resolver).
func newVisitor(
	records chan<- Record,
	bs blob.Store,
	clock timeutil.Clock) (v dag.Visitor) {
	panic("TODO")
}

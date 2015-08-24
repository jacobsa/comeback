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

package save

import (
	"errors"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/graph"
)

// Create a graph.Visitor which does the following, for each node n of type
// *fsNode:
//
// 1.  If n.Scores == nil, use the supplied blob store to fill in the
//     appropriate scores.
//
// 2.  Write n to nodesProcessed.
//
func newVisitor(
	blobStore blob.Store,
	nodesProcessed chan<- *fsNode) (v graph.Visitor, err error) {
	err = errors.New("TODO")
	return
}

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

	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/timeutil"

	"golang.org/x/net/context"
)

// Attempt to fill in fsNode.Scores fields for files arriving on nodesIn based
// on the contents of a state.ScoreMap.
func consultScoreMap(
	ctx context.Context,
	scoreMap state.ScoreMap,
	clock timeutil.Clock,
	nodesIn <-chan *fsNode,
	nodesOut chan<- *fsNode) (err error) {
	// TODO(jacobsa): Make sure to consult score_map_saver.go. We don't need the
	// bit that talks to the blob store (added in abd1800) if we kill blob store
	// internal asynchronicity, though.
	err = errors.New("TODO")
	return
}

// For each incoming file node n that consultScoreMap did not mark as having
// hit in its score map, update the score map based on n.Scores.
func updateScoreMap(
	ctx context.Context,
	scoreMap state.ScoreMap,
	nodes <-chan *fsNode) (err error) {
	// TODO(jacobsa): Make sure to consult score_map_saver.go.
	err = errors.New("TODO")
	return
}

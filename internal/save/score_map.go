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
	"os"
	"syscall"
	"time"

	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/comeback/internal/sys"
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

// Return an appropriate score map key for the node, or nil if the score map
// should not be used.
func makeScoreMapKey(
	node *fsNode,
	clock timeutil.Clock) (key *state.ScoreMapKey) {
	fi := node.Info

	// Skip non-files.
	if fi.Mode()&os.ModeType != 0 {
		return
	}

	// If the mtime of the file is not far enough in the past, we don't want to
	// do any fancy caching, for fear of race conditions.
	const minElapsed = 5 * time.Minute
	if clock.Now().Sub(fi.ModTime()) < minElapsed {
		return
	}

	// Return an appropriate key.
	stat := fi.Sys().(*syscall.Stat_t)
	key = &state.ScoreMapKey{
		Path:        node.RelPath,
		Permissions: fi.Mode() & os.ModePerm,
		Uid:         sys.UserId(stat.Uid),
		Gid:         sys.GroupId(stat.Gid),
		MTime:       fi.ModTime(),
		Inode:       stat.Ino,
		Size:        uint64(fi.Size()),
	}

	return
}

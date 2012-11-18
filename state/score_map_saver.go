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

package state

import (
	"fmt"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"time"
)

// Create a file saver that first attempts to read scores from the supplied
// source map, only calling the wrapped saver when the map doesn't have an
// answer.
//
// Any scores seen, regardless of their source, are written to the sink map.
func NewScoreMapFileSaver(
	sourceMap ScoreMap,
	sinkMap ScoreMap,
	fileSystem fs.FileSystem,
	wrapped backup.FileSaver,
) (s backup.FileSaver) {
	return newScoreMapFileSaver(
		sourceMap,
		sinkMap,
		fileSystem,
		wrapped,
		time.Now,
	)
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

// Like NewScoreMapFileSaver, but allows injecting a time.Now()-like function.
func newScoreMapFileSaver(
	sourceMap ScoreMap,
	sinkMap ScoreMap,
	fileSystem fs.FileSystem,
	wrapped backup.FileSaver,
	now func() time.Time,
) (s backup.FileSaver) {
	return &scoreMapFileSaver{
		sourceMap,
		sinkMap,
		fileSystem,
		wrapped,
		now,
	}
}

type scoreMapFileSaver struct {
	sourceMap  ScoreMap
	sinkMap    ScoreMap
	fileSystem fs.FileSystem
	wrapped    backup.FileSaver
	now        func() time.Time
}

func (s *scoreMapFileSaver) Save(path string) (scores []blob.Score, err error) {
	// Stat the file.
	entry, err := s.fileSystem.Stat(path)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	// If the mtime of the file is not far enough in the past, we don't want to
	// do any fancy caching, for fear of race conditions.
	const minElapsed = 5 * time.Minute
	if s.now().Sub(entry.MTime) < minElapsed {
		return s.wrapped.Save(path)
	}

	// Whatever we do, make sure that we insert any result we find into the sink
	// map.
	mapKey := ScoreMapKey{
		Path:        path,
		Permissions: entry.Permissions,
		Uid:         entry.Uid,
		Gid:         entry.Gid,
		MTime:       entry.MTime,
		Inode:       entry.Inode,
		Size:        entry.Size,
	}

	defer func() {
		if err == nil {
			s.sinkMap.Set(mapKey, scores)
		}
	}()

	// Do we have anything interesting in the map?
	if scores = s.sourceMap.Get(mapKey); scores != nil {
		return
	}

	// Pass on the request.
	return s.wrapped.Save(path)
}

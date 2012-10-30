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
)

// Create a file saver that first attempts to read scores from the supplied
// map, only calling the wrapped saver when the map doesn't have an answer.
func NewMapReadingFileSaver(
	scoreMap ScoreMap,
	fileSystem fs.FileSystem,
	wrapped backup.FileSaver,
) (s backup.FileSaver) {
	return &mapReadingFileSaver{
		scoreMap,
		fileSystem,
		wrapped,
	}
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type mapReadingFileSaver struct {
	scoreMap ScoreMap
	fileSystem fs.FileSystem
	wrapped backup.FileSaver
}

func (s *mapReadingFileSaver) Save(path string) (scores []blob.Score, err error) {
	err = fmt.Errorf("TODO")
	return
}

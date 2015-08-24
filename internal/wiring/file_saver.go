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

package wiring

import (
	"fmt"

	"github.com/jacobsa/comeback/internal/backup"
	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/state"
)

// Create a file saver that uses the supplied file system and blob store.
//
// chunkSize is the size of chunks into which large files are split.
//
// scoresForFiles is a cache from file system info to the scores that were seen
// at the time that file was stat'd, to be used in saving the work of reading
// file contents each time. It will be updated by the file saver.
func makeFileSaver(
	bs blob.Store,
	fs fs.FileSystem,
	chunkSize int,
	scoresForFiles state.ScoreMap) (fileSaver backup.FileSaver, err error) {
	// Write chunks to the blob store.
	fileSaver, err = backup.NewFileSaver(bs, chunkSize, fs)
	if err != nil {
		err = fmt.Errorf("NewFileSaver: %v", err)
		return
	}

	// Avoid computing scores when unnecessary.
	fileSaver = state.NewScoreMapFileSaver(scoresForFiles, bs, fs, fileSaver)

	return
}
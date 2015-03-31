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

package main

import (
	"log"
	"sync"

	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/state"
)

var g_fileSaverOnce sync.Once
var g_fileSaver backup.FileSaver

func initFileSaver() {
	var err error

	blobStore := getBlobStore()
	fileSystem := getFileSystem()
	stateStruct := getState()

	// Write chunks to the blob store.
	const chunkSize = 1 << 24 // 16 MiB

	g_fileSaver, err = backup.NewFileSaver(
		blobStore,
		chunkSize,
		fileSystem,
	)

	if err != nil {
		log.Fatalln("Creating file saver:", err)
	}

	// Avoid computing scores when unnecessary.
	g_fileSaver = state.NewScoreMapFileSaver(
		stateStruct.ScoresForFiles,
		blobStore,
		fileSystem,
		g_fileSaver)
}

func getFileSaver() backup.FileSaver {
	g_fileSaverOnce.Do(initFileSaver)
	return g_fileSaver
}

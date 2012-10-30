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
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/concurrent"
	"github.com/jacobsa/comeback/state"
	"log"
	"runtime"
	"sync"
)

var g_fileSaverOnce sync.Once
var g_fileSaver backup.FileSaver

func initFileSaver() {
	var err error

	blobStore := getBlobStore()
	fileSystem := getFileSystem()
	stateStruct := getState()

	// Set up parallelism. Leave one CPU alone, if possible.
	numCPUs := runtime.NumCPU()

	numFileSaverWorkers := numCPUs
	if numCPUs > 1 {
		numFileSaverWorkers--
	}

	runtime.GOMAXPROCS(numFileSaverWorkers)

	// Write chunks to the blob store in parallel.
	const chunkSize = 1 << 24 // 16 MiB

	g_fileSaver, err = backup.NewFileSaver(
		blobStore,
		chunkSize,
		fileSystem,
		concurrent.NewExecutor(numFileSaverWorkers),
	)

	if err != nil {
		log.Fatalln("Creating file saver:", err)
	}

	// Avoid computing scores when unnecessary. Save all returned scores to a new
	// map in the state object.
	sourceMap := stateStruct.ScoresForFiles
	sinkMap := state.NewScoreMap()
	stateStruct.ScoresForFiles = sinkMap

	g_fileSaver = state.NewScoreMapFileSaver(
		sourceMap,
		sinkMap,
		fileSystem,
		g_fileSaver)
}

func getFileSaver() backup.FileSaver {
	g_fileSaverOnce.Do(initFileSaver)
	return g_fileSaver
}

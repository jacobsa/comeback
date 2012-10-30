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
	"log"
	"sync"
)

var g_dirSaverOnce sync.Once
var g_dirSaver backup.DirectorySaver

func initDirSaver() {
	var err error
	blobStore := getBlobStore()
	fileSystem := getFileSystem()
	fileSaver := getFileSaver()

	// Create the directory saver.
	g_dirSaver, err = backup.NewDirectorySaver(
		blobStore,
		fileSystem,
		fileSaver)

	if err != nil {
		log.Fatalln("Creating directory saver:", err)
	}
}

func getDirSaver() backup.DirectorySaver {
	g_dirSaverOnce.Do(initDirSaver)
	return g_dirSaver
}
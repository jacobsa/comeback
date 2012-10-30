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
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/sys"
	"log"
	"sync"
)

var g_fileSystemOnce sync.Once
var g_fileSystem fs.FileSystem

func initFileSystem() {
	// Create a user registry.
	userRegistry, err := sys.NewUserRegistry()
	if err != nil {
		log.Fatalln("Creating user registry:", err)
	}

	// Create a group registry.
	groupRegistry, err := sys.NewGroupRegistry()
	if err != nil {
		log.Fatalln("Creating group registry:", err)
	}

	// Create the file system.
	g_fileSystem, err = fs.NewFileSystem(userRegistry, groupRegistry)
	if err != nil {
		log.Fatalln("Creating file system:", err)
	}
}

func getFileSystem() fs.FileSystem {
	g_fileSystemOnce.Do(initFileSystem)
	return g_fileSystem
}

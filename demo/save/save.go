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
	"flag"
	"fmt"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob/disk"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/sys"
	"io/ioutil"
	"log"
	"os"
)

var g_configFile = flag.String("config", "", "Path to config file.")

func main() {
	var err error
	flag.Parse()

	// Attempt to read the user's config data.
	if *g_configFile == "" {
		fmt.Println("You must set -config.")
		os.Exit(1)
	}

	configData, err := ioutil.ReadFile(*g_configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	// Create a user registry.
	userRegistry, err := sys.NewUserRegistry()
	if err != nil {
		log.Fatalf("Creating user registry: %v", err)
	}

	// Create a group registry.
	groupRegistry, err := sys.NewGroupRegistry()
	if err != nil {
		log.Fatalf("Creating group registry: %v", err)
	}

	// Create a file system.
	fileSystem, err := fs.NewFileSystem(userRegistry, groupRegistry)
	if err != nil {
		log.Fatalf("Creating file system: %v", err)
	}

	// Create the blob store.
	blobStore, err := disk.NewDiskBlobStore("/tmp/blobs", fileSystem)
	if err != nil {
		log.Fatalf("Creating store: %v", err)
	}

	// Create the file saver.
	fileSaver, err := backup.NewFileSaver(blobStore, 1<<24)
	if err != nil {
		log.Fatalf("Creating file saver: %v", err)
	}

	// Create a directory saver.
	dirSaver, err := backup.NewDirectorySaver(
		blobStore,
		fileSystem,
		fileSaver)

	if err != nil {
		log.Fatalf("Creating directory saver: %v", err)
	}

	// Save a directory.
	score, err := dirSaver.Save("/Volumes/Src", "")
	if err != nil {
		log.Fatalf("Saving: %v", err)
	}

	// Print the score.
	fmt.Printf("Score: %s\n", score.Hex())
}

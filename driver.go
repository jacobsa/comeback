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
	"fmt"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/disk"
	"github.com/jacobsa/comeback/fs"
	"log"
)

func main() {
	// Create the blob store.
	blobStore, err := disk.NewBlobStore("/tmp/blobs")
	if err != nil {
		log.Fatalf("Creating store: %v", err)
	}

	// Create the file saver.
  fileSaver, err := backup.NewFileSaver(blobStore)
	if err != nil {
		log.Fatalf("Creating file saver: %v", err)
	}

	// Create a directory saver.
	dirSaver, err := backup.NewDirectorySaver(
		blobStore,
		fs.NewFileSystem(),
		fileSaver,
		nil)

	if err != nil {
		log.Fatalf("Creating directory saver: %v", err)
	}

	// Create another level.
	dirSaver, err = backup.NewDirectorySaver(
		blobStore,
		fs.NewFileSystem(),
		fileSaver,
		dirSaver)

	if err != nil {
		log.Fatalf("Creating directory saver: %v", err)
	}

	// Save a directory.
	score, err := dirSaver.Save("/Users/jacobsa/tmp")
	if err != nil {
		log.Fatalf("Saving: %v", err)
	}

	// Print the score.
	fmt.Printf("Score: %x\n", score.Sha1Hash())
}

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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/disk"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr"
	"log"
	"os"
	"path"
)

const (
	gTarget = "/tmp/restore_target"
)

var blobStore blob.Store

func fromHexHash(hexHash string) (blob.Score, error) {
	return nil, fmt.Errorf("TODO: fromHexHash")
}

// Restore the file whose contents are described by the referenced blobs to the
// supplied target, whose parent must already exist.
func restoreFile(target string, scores []blob.Score) error {
	return fmt.Errorf("TODO: restoreFile")
}

// Restore the directory whose contents are described by the referenced blob to
// the supplied target, which must already exist.
func restoreDir(target string, score blob.Score) error {
	// Load the appropriate blob.
	blob, err := blobStore.Load(score)
	if err != nil {
		return fmt.Errorf("Loading blob: %v", err)
	}

	// Parse its contents.
	entries, err := repr.Unmarshal(blob)
	if err != nil {
		return fmt.Errorf("Parsing blob: %v", err)
	}

	// Deal with each entry.
	for _, entry := range entries {
		entryPath := path.Join(target, entry.Name)

		// Switch on type.
		switch entry.Type {
		case fs.TypeFile:
			err = restoreFile(entryPath, entry.Scores)
			if err != nil {
				return err
			}

		case fs.TypeDirectory:
			if len(entry.Scores) != 1 {
				return fmt.Errorf("Wrong number of scores: %v", entry)
			}

			if err = os.Mkdir(entryPath, 0700); err != nil {
				return err
			}

			if err = restoreDir(entryPath, entry.Scores[0]); err != nil {
				return err
			}

		default:
			return fmt.Errorf("Don't know how to deal with entry: %v", entry)
		}
	}

	return nil
}

func main() {
	var err error

	// Create the blob store.
	blobStore, err = disk.NewBlobStore("/tmp/blobs")
	if err != nil {
		log.Fatalf("Creating store: %v", err)
	}

	// Parse the score.
	score, err := fromHexHash("ee905ccc3416d120c962dc983023c28401ae39ae")
	if err != nil {
		log.Fatalf("Parsing score: %v", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(gTarget)
	if err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}

	// Create the target.
	err = os.Mkdir("/tmp/restore_target", 0700)
	if err != nil {
		log.Fatalf("Mkdir: %v", err)
	}

	// Attempt a restore.
	err = restoreDir(gTarget, score)
	if err != nil {
		log.Fatalf("Restoring: %v", err)
	}
}

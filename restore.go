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
	"log"
	"os"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/restore"

	"golang.org/x/net/context"
)

var cmdRestore = &Command{
	Name: "restore",
	Run:  runRestore,
}

func runRestore(ctx context.Context, args []string) (err error) {
	// Extract and parse arguments.
	if len(args) != 2 {
		err = fmt.Errorf("Usage: %s restore dst_dir score", os.Args[0])
		return
	}

	dstDir := args[0]
	score, err := blob.ParseHexScore(args[1])
	if err != nil {
		err = fmt.Errorf("ParseHexScore(%q): %v", args[1], err)
		return
	}

	// Grab dependencies.
	blobStore := getBlobStore(ctx)

	// Make sure the target doesn't exist.
	err = os.RemoveAll(dstDir)
	if err != nil {
		err = fmt.Errorf("os.RemoveAll: %v", err)
		return
	}

	// Create the destination.
	err = os.Mkdir(dstDir, 0700)
	if err != nil {
		err = fmt.Errorf("os.Mkdir: %v", err)
		return
	}

	// Attempt a restore.
	err = restore.Restore(
		ctx,
		dstDir,
		score,
		blobStore,
		log.New(os.Stderr, "Restore progress: ", 0),
	)

	if err != nil {
		err = fmt.Errorf("Restoring: %v", err)
		return
	}

	log.Printf("Successfully restored to ", dstDir)
	return
}

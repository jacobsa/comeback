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

package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/comebackfs"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/comeback/internal/wiring"
)

var cmdMount = &Command{
	Name: "mount",
}

func init() {
	cmdMount.Run = runMount // Break flag-related dependency loop.
}

func doMount(args []string) (err error) {
	// Grab dependencies.
	bucket := getBucket()
	crypter := getCrypter()

	// Check usage.
	if len(args) > 1 {
		err = fmt.Errorf("Usage: %s [score]", os.Args[0])
		return
	}

	// Figure out which score to mount.
	var score blob.Score
	if len(args) > 0 {
		score, err = blob.ParseHexScore(args[0])
		if err != nil {
			err = fmt.Errorf("ParseHexScore(%q): %v", args[0], err)
			return
		}
	} else {
		r := getRegistry()

		// List jobs.
		var jobs []registry.CompletedJob
		jobs, err = r.ListBackups()
		if err != nil {
			err = fmt.Errorf("ListBackups: %v", err)
			return
		}

		if len(jobs) == 0 {
			err = errors.New("No completed jobs found.")
			return
		}

		// Find the job with the newest start time.
		j := jobs[0]
		for _, candidate := range jobs {
			if j.StartTime.Before(candidate.StartTime) {
				j = candidate
			}
		}

		score = j.Score
	}

	log.Printf("Mounting score %s.", score.Hex())

	// Create the blob store.
	blobStore, err := wiring.MakeBlobStore(
		bucket,
		crypter,
		util.NewStringSet())

	// Create the file system.
	_, err = comebackfs.NewFileSystem(score, blobStore)
	if err != nil {
		err = fmt.Errorf("NewFileSystem: %v", err)
		return
	}

	err = errors.New("TODO")
	return
}

func runMount(args []string) {
	err := doMount(args)
	if err != nil {
		log.Fatalln(err)
	}
}

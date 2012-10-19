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
	"os"
	"strconv"
)

var cmdRestore = &Command{
	Name: "restore",
}

var g_jobIdStr = cmdRestore.Flags.String("job_id", "", "The job ID to restore.")
var g_target = cmdRestore.Flags.String("target", "", "The target directory.")

func init() {
	cmdRestore.Run = runRestore  // Break flag-related dependency loop.
}

func runRestore(args []string) {
	// Parse the job ID.
	if len(*g_jobIdStr) != 16 {
		log.Fatalln("You must set the -job_id flag.")
	}

	jobId, err := strconv.ParseUint(*g_jobIdStr, 16, 64)
	if err != nil {
		log.Fatalln("Invalid job ID:", *g_jobIdStr)
	}

	// Check the target.
	if *g_target == "" {
		log.Fatalln("You must set the -target flag.")
	}

	// Grab dependencies.
	dirRestorer := getDirRestorer()
	reg := getRegistry()

	// Find the requested job.
	job, err := reg.FindBackup(jobId)
	if err != nil {
		log.Fatalln("FindBackup:", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(*g_target)
	if err != nil {
		log.Fatalln("os.RemoveAll:", err)
	}

	// Create the target.
	err = os.Mkdir(*g_target, 0700)
	if err != nil {
		log.Fatalln("os.Mkdir:", err)
	}

	// Attempt a restore.
	err = dirRestorer.RestoreDirectory(
		job.Score,
		*g_target,
		"",
	)

	if err != nil {
		log.Fatalln("Restoring:", err)
	}

	log.Println("Successfully restored to target:", *g_target)
}

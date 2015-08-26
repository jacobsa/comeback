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
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
)

var cmdRestore = &Command{
	Name: "restore",
}

var g_jobTime = cmdRestore.Flags.String(
	"job_time", "", "The start time of the job to restore.")
var g_target = cmdRestore.Flags.String("target", "", "The target directory.")

func init() {
	cmdRestore.Run = runRestore // Break flag-related dependency loop.
}

func runRestore(ctx context.Context, args []string) (err error) {
	// Parse the job start time.
	if len(*g_jobTime) == 0 {
		err = errors.New("You must set the --job_time flag.")
		return
	}

	startTime, err := time.Parse(time.RFC3339Nano, *g_jobTime)
	if err != nil {
		err = fmt.Errorf("Parsing --job_time: %v", err)
		return
	}

	// Check the target.
	if *g_target == "" {
		err = errors.New("You must set the -target flag.")
		return
	}

	// Grab dependencies. Make sure to get the registry first, because it takes
	// less time to construct.
	reg := getRegistry()
	dirRestorer := getDirRestorer()

	// Find the requested job.
	job, err := reg.FindBackup(startTime)
	if err != nil {
		err = fmt.Errorf("FindBackup: %v", err)
		return
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(*g_target)
	if err != nil {
		err = fmt.Errorf("os.RemoveAll: %v", err)
		return
	}

	// Create the target.
	err = os.Mkdir(*g_target, 0700)
	if err != nil {
		err = fmt.Errorf("os.Mkdir: %v", err)
		return
	}

	// Attempt a restore.
	err = dirRestorer.RestoreDirectory(
		job.Score,
		*g_target,
		"",
	)

	if err != nil {
		err = fmt.Errorf("Restoring: %v", err)
		return
	}

	log.Println("Successfully restored to target:", *g_target)
	return
}

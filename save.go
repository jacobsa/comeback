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
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/config"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/comeback/internal/save"
	"github.com/jacobsa/timeutil"
)

var cmdSave = &Command{
	Name: "save",
}

var g_discardScoreCache = cmdSave.Flags.Bool(
	"discard_score_cache",
	false,
	"If set, always recompute file hashes; don't rely on stat info.",
)

var fListOnly = cmdSave.Flags.Bool(
	"list_only",
	false,
	"If set, list the files that would be backed up but do nothing further.")

func init() {
	cmdSave.Run = runSave // Break flag-related dependency loop.
}

func saveStatePeriodically(
	ctx context.Context,
	c <-chan time.Time) {
	for _ = range c {
		log.Println("Writing out state file.")
		saveState(ctx)
	}
}

func doList(ctx context.Context, job *config.Job) (err error) {
	err = save.List(
		ctx,
		os.Stdout,
		job.BasePath,
		job.Excludes)

	if err != nil {
		err = fmt.Errorf("save.List: %v", err)
		return
	}

	return
}

func runSave(ctx context.Context, args []string) (err error) {
	cfg := getConfig()

	// Extract arguments.
	if len(args) != 1 {
		err = fmt.Errorf("Usage: %s save job_name", os.Args[0])
		return
	}

	jobName := args[0]

	// Look for the specified job.
	job, ok := cfg.Jobs[jobName]
	if !ok {
		err = fmt.Errorf("Unknown job: %q", jobName)
		return
	}

	// Special case: visit the file system only if --list_only is set.
	if *fListOnly {
		err = doList(ctx, job)
		if err != nil {
			err = fmt.Errorf("doList: %v", err)
			return
		}

		return
	}

	// Grab dependencies. Make sure to get the registry first, because otherwise
	// the user will have to wait for bucket keys to be listed before being
	// prompted for a crypto password.
	//
	// Make sure to do this before setting up state saving below, because these
	// calls may modify the state struct.
	reg := getRegistry(ctx)
	blobStore := getBlobStore(ctx)
	state := getState(ctx)
	clock := timeutil.RealClock()

	// Periodically save state.
	const saveStatePeriod = 15 * time.Second
	saveStateTicker := time.NewTicker(saveStatePeriod)
	go saveStatePeriodically(ctx, saveStateTicker.C)

	// Choose a start time for the job.
	startTime := clock.Now()

	// Call the saving pipeline.
	score, err := save.Save(
		ctx,
		job.BasePath,
		job.Excludes,
		state.ScoresForFiles,
		blobStore,
		log.New(os.Stderr, "Save progress: ", 0),
		clock)

	if err != nil {
		err = fmt.Errorf("save.Save: %v", err)
		return
	}

	// Register the successful backup.
	completedJob := registry.CompletedJob{
		StartTime: startTime,
		Name:      jobName,
		Score:     score,
	}

	err = reg.RecordBackup(ctx, completedJob)
	if err != nil {
		err = fmt.Errorf("RecordBackup: %v", err)
		return
	}

	log.Printf(
		"Successfully backed up with score %v. Start time: %v\n",
		score.Hex(),
		startTime.UTC())

	// Store state for next time.
	saveStateTicker.Stop()
	log.Println("Writing out final state file...")
	saveState(ctx)

	return
}

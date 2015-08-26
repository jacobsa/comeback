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

	"github.com/jacobsa/comeback/internal/config"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/comeback/internal/save"
)

var cmdSave = &Command{
	Name: "save",
}

var g_jobName = cmdSave.Flags.String(
	"job",
	"",
	"Job name within the config file.")

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

	// Look for the specified job.
	if *g_jobName == "" {
		err = errors.New("You must set the -job flag.")
		return
	}

	job, ok := cfg.Jobs[*g_jobName]
	if !ok {
		err = fmt.Errorf("Unknown job: %q", *g_jobName)
		return
	}

	// Special case: visit the file system only if --list_only is set.
	//
	// TODO(jacobsa): Integrate this into the pipeline when it exists. See issue
	// #21.
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
	dirSaver := getDirSaver(ctx)

	// Periodically save state.
	const saveStatePeriod = 15 * time.Second
	saveStateTicker := time.NewTicker(saveStatePeriod)
	go saveStatePeriodically(ctx, saveStateTicker.C)

	// Choose a start time for the job.
	startTime := time.Now()

	// Call the directory saver.
	score, err := dirSaver.Save(job.BasePath, "", job.Excludes)
	if err != nil {
		err = fmt.Errorf("dirSaver.Save: %v", err)
		return
	}

	// Ensure the backup is durable.
	err = dirSaver.Flush()
	if err != nil {
		err = fmt.Errorf("dirSaver.Flush: %v", err)
		return
	}

	// Register the successful backup.
	completedJob := registry.CompletedJob{
		StartTime: startTime,
		Name:      *g_jobName,
		Score:     score,
	}

	err = reg.RecordBackup(completedJob)
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

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
	"github.com/jacobsa/comeback/registry"
	"log"
	"time"
)

var cmdSave = &Command{
	Name: "save",
}

var g_jobName = cmdSave.Flags.String("job", "", "Job name within the config file.")

func init() {
	cmdSave.Run = runSave  // Break flag-related dependency loop.
}

func runSave(args []string) {
	cfg := getConfig()

	// Look for the specified job.
	if *g_jobName == "" {
		log.Fatalln("You must set the -job flag.")
	}

	job, ok := cfg.Jobs[*g_jobName]
	if !ok {
		log.Fatalln("Unknown job:", *g_jobName)
	}

	// Grab dependencies.
	dirSaver := getDirSaver()
	reg := getRegistry()

	// Choose a start time for the job.
	startTime := time.Now()

	// Call the directory saver.
	score, err := dirSaver.Save(job.BasePath, "", job.Excludes)
	if err != nil {
		log.Fatalln(err)
	}

	// Register the successful backup.
	randSrc := getRandSrc()
	completedJob := registry.CompletedJob{
		Id:        randUint64(randSrc),
		Name:      *g_jobName,
		StartTime: startTime,
		Score:     score,
	}

	if err = reg.RecordBackup(completedJob); err != nil {
		log.Fatalln("Recoding to registry:", err)
	}

	log.Printf("Successfully backed up. ID: %16x\n", completedJob.Id)
}

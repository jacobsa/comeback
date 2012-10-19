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
)

var cmdList = &Command{
	Name: "list",
	Run: runList,
}

func runList(args []string) {
	// Ask the registry for a list.
	registry := getRegistry()
	jobs, err := registry.ListRecentBackups()
	if err != nil {
		log.Fatalln("Listing recent backups:", err)
	}

	// Print each.
	log.Println("")
	log.Println("")
	log.Println("Recent backups:")
	log.Println("")
	log.Printf(
		"  %-16s   %-40s   %-29s   %s\n",
		"ID",
		"JOB NAME",
		"START TIME",
		"SCORE",
	)

	for _, job := range jobs {
		log.Printf(
			"  %16x   %-40s   %-29s   %s\n",
			job.Id,
			job.Name,
			job.StartTime,
			job.Score.Hex(),
		)
	}
}

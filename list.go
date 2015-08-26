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
	"time"

	"golang.org/x/net/context"
)

var cmdList = &Command{
	Name: "list",
	Run:  runList,
}

func runList(ctx context.Context, args []string) (err error) {
	// Ask the registry for a list.
	registry := getRegistry()
	jobs, err := registry.ListBackups()
	if err != nil {
		err = fmt.Errorf("ListBackups: %v", err)
		return
	}

	// Print each.
	log.Println("")
	log.Println("")
	log.Println("Previous backups:")
	log.Println("")
	log.Printf(
		"  %-38s   %-40s   %s\n",
		"START TIME",
		"JOB NAME",
		"SCORE",
	)

	for _, job := range jobs {
		log.Printf(
			"  %-38s   %-40s   %s\n",
			job.StartTime.Format(time.RFC3339Nano),
			job.Name,
			job.Score.Hex(),
		)
	}

	return
}

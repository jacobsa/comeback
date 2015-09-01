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
	"os"
	"text/tabwriter"
	"time"

	"golang.org/x/net/context"
)

var cmdList = &Command{
	Name: "list",
	Run:  runList,
}

func runList(ctx context.Context, args []string) (err error) {
	// Ask the registry for a list.
	registry := getRegistry(ctx)
	jobs, err := registry.ListBackups(ctx)
	if err != nil {
		err = fmt.Errorf("ListBackups: %v", err)
		return
	}

	// Print each.
	const minwidth = 0
	const tabwidth = 8
	const padding = 4
	const padchar = '\t'
	const flags = 0

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, minwidth, tabwidth, padding, padchar, flags)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Start time\tJob name\tScore")

	for _, job := range jobs {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\n",
			job.StartTime.Format(time.RFC3339Nano),
			job.Name,
			job.Score.Hex(),
		)
	}

	w.Flush()

	return
}

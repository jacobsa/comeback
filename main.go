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
	"flag"
	"io"
	"log"
	"os"
	"path"

	"github.com/jacobsa/gcloud/gcs"
)

// The set of commands supported by the tool.
var commands = []*Command{
	cmdDeleteGarbage,
	cmdGC,
	cmdList,
	cmdRestore,
	cmdSave,
	cmdVerify,
}

func main() {
	flag.Parse()

	// Set up bare logging output.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Set up the GCS log.
	gcsLog, err := os.OpenFile(
		path.Join("/Users/jacobsa/.comeback.gcs.log"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	if err != nil {
		log.Fatalf("OpenFile: %v", err)
		return
	}

	defer gcsLog.Close()

	gcs.SetLogger(log.New(
		gcsLog,
		"",
		log.LstdFlags|log.Lmicroseconds|log.Lshortfile))

	// Set up saving of default logging output.
	logLog, err := os.OpenFile(
		path.Join("/Users/jacobsa/.comeback.log.log"),
		os.O_RDWR|os.O_APPEND|os.O_CREATE,
		0600)

	if err != nil {
		log.Fatalf("OpenFile: %v", err)
		return
	}

	defer logLog.Close()

	log.SetOutput(io.MultiWriter(os.Stderr, logLog))

	// We get the command name.
	args := flag.Args()
	if len(args) < 1 {
		log.Println("Missing command name. Choices are:")
		for _, cmd := range commands {
			log.Printf("  %s\n", cmd.Name)
		}

		os.Exit(1)
	}

	cmdName := args[0]

	// Find and run the appropriate command.
	for _, cmd := range commands {
		if cmd.Name == cmdName {
			cmd.Flags.Parse(args[1:])
			args = cmd.Flags.Args()
			cmd.Run(args)
			return
		}
	}

	log.Fatalln("Unknown command:", cmdName)
}

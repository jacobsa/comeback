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
	"log"
)

// The set of commands supported by the tool.
var commands = []*Command{
	cmdList,
	cmdSave,
	cmdRestore,
}

func main() {
	flag.Parse()

	// Set up bare logging output.
	log.SetFlags(0)

	// We get the command name.
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalln("Missing command name.")
	}

	cmdName := args[0]

	// Find and run the appropriate command.
	for _, cmd := range commands {
		if cmd.Name == cmdName {
			cmd.Run(args)
			return
		}
	}

	log.Fatalln("Unknown command:", cmdName)
}

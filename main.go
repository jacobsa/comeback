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
	"fmt"
	"log"
	"os"

	"golang.org/x/net/context"
)

// The set of commands supported by the tool.
var commands = []*Command{
	cmdDeleteGarbage,
	cmdGC,
	cmdList,
	cmdMount,
	cmdRestore,
	cmdSave,
	cmdVerify,
}

func runCmd(
	ctx context.Context,
	cmdName string,
	cmdArgs []string) (err error) {
	// Find and run the appropriate command.
	for _, cmd := range commands {
		if cmd.Name == cmdName {
			cmd.Flags.Parse(cmdArgs)
			err = cmd.Run(ctx, cmd.Flags.Args())
			return
		}
	}

	err = fmt.Errorf("Unknown command: %q", cmdName)
	return
}

func main() {
	flag.Parse()

	// Set up bare logging output.
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	// Find the command name.
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Missing command name. Choices are:")
		for _, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  %s\n", cmd.Name)
		}

		os.Exit(1)
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	// Call through.
	err := runCmd(context.Background(), cmdName, cmdArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

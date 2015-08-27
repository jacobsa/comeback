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
	"runtime"
	"runtime/pprof"
	"syscall"

	"golang.org/x/net/context"
)

var fProfile = flag.Bool("profile", false, "Write pprof profiles to /tmp.")

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Raise the rlimit for number of open files to a sane value.
func raiseRlimit() (err error) {
	// Find the current limit.
	var rlimit syscall.Rlimit
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		err = fmt.Errorf("Getrlimit: %v", err)
		return
	}

	// Raise it to the hard limit.
	rlimit.Cur = rlimit.Max
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		err = fmt.Errorf("Setrlimit: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Commands
////////////////////////////////////////////////////////////////////////

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
	// Enable profiling, if requested.
	if *fProfile {
		// Memory
		defer writeMemProfile("/tmp/mem.pprof")

		// CPU
		var f *os.File
		f, err = os.Create("/tmp/cpu.pprof")
		if err != nil {
			err = fmt.Errorf("Create: %v", err)
			return
		}

		defer f.Close()

		// Profile.
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

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

////////////////////////////////////////////////////////////////////////
// Profiling
////////////////////////////////////////////////////////////////////////

func writeMemProfile(path string) (err error) {
	// Trigger a garbage collection to get up to date information (cf.
	// https://goo.gl/aXVQfL).
	runtime.GC()

	// Open the file.
	var f *os.File
	f, err = os.Create(path)
	if err != nil {
		err = fmt.Errorf("Create: %v", err)
		return
	}

	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		}
	}()

	// Dump to the file.
	err = pprof.Lookup("heap").WriteTo(f, 0)
	if err != nil {
		err = fmt.Errorf("WriteTo: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// main
////////////////////////////////////////////////////////////////////////

func main() {
	flag.Parse()

	// Set up bare logging output.
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	// Attempt to avoid "too many open files" errors.
	err := raiseRlimit()
	if err != nil {
		log.Fatal(err)
	}

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
	err = runCmd(context.Background(), cmdName, cmdArgs)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

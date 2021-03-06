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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"syscall"
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
		// Write a memory profile.
		defer writeMemProfile("/tmp/mem.pprof")

		// First trigger a garbage collection to get up to date information (cf.
		// https://goo.gl/aXVQfL).
		defer runtime.GC()

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

func init() {
	// Listen for SIGUSR1, dumping a memory profile when it's received.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)

		for range c {
			var ms runtime.MemStats

			runtime.ReadMemStats(&ms)
			log.Printf("Pre-GC mem stats:\n%s", formatMemStats(&ms))

			// Trigger a garbage collection to get up to date information (cf.
			// https://goo.gl/aXVQfL).
			runtime.GC()

			runtime.ReadMemStats(&ms)
			log.Printf("Post-GC mem stats:\n%s", formatMemStats(&ms))

			const path = "/tmp/mem.pprof"
			err := writeMemProfile(path)
			if err != nil {
				log.Printf("Error writing profile: %v", err)
			} else {
				log.Printf("Profile written to %s", path)
			}
		}
	}()
}

func formatMemStats(ms *runtime.MemStats) string {
	fields := []string{
		"Alloc",
		"TotalAlloc",
		"Sys",
		"HeapAlloc",
		"HeapSys",
		"HeapIdle",
		"HeapInuse",
		"HeapReleased",
	}

	var lines []string
	v := reflect.ValueOf(*ms)
	for _, f := range fields {
		fv := v.FieldByName(f)
		if !fv.IsValid() {
			panic(fmt.Sprintf("bad field: %q", f))
		}

		lines = append(lines, fmt.Sprintf("  %12s: %s", f, formatBytes(fv.Uint())))
	}

	return strings.Join(lines, "\n")
}

func formatBytes(b uint64) string {
	var val float64
	var unit string
	switch {
	case b >= 1<<30:
		val = float64(b) / (1 << 30)
		unit = "GiB"

	case b >= 1<<20:
		val = float64(b) / (1 << 20)
		unit = "MiB"

	case b >= 1<<10:
		val = float64(b) / (1 << 10)
		unit = "KiB"

	default:
		val = float64(b)
		unit = "bytes"
	}

	return fmt.Sprintf("%6.2f %s", val, unit)
}

////////////////////////////////////////////////////////////////////////
// main
////////////////////////////////////////////////////////////////////////

func main() {
	flag.Parse()

	// We naturally have a large working set, since we need to buffer lots of
	// data in flight to GCS over a high latency link. Ensure that we don't waste
	// a ton of memory on heap garbage.
	if _, gogcSet := os.LookupEnv("GOGC"); !gogcSet {
		debug.SetGCPercent(25)
	}

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

// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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
	"os/user"
	"strconv"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/comebackfs"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/daemonize"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
	"github.com/kardianos/osext"
)

var cmdMount = &Command{
	Name: "mount",
}

var fDebugFuse = cmdMount.Flags.Bool(
	"debug_fuse",
	false,
	"Enable fuse debug logging.")

func init() {
	cmdMount.Run = runMount // Break flag-related dependency loop.
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Return the UID and GID of the current process.
func currentUser() (uid uint32, gid uint32, err error) {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	// Parse the UID.
	uid64, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		err = fmt.Errorf("Parsing UID (%s): %v", user.Uid, err)
		return
	}

	// Parse the GID.
	gid64, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		err = fmt.Errorf("Parsing GID (%s): %v", user.Gid, err)
		return
	}

	uid = uint32(uid64)
	gid = uint32(gid64)

	return
}

// When the user runs `comeback mount`, the binary re-executes itself using the
// daemonize package. This environment variable is set when invoking the daemon
// so that it can tell itself apart from the foreground process.
const daemonEnvVar = "COMEBACK_DAEMON"

func isDaemon() (b bool) {
	_, b = os.LookupEnv(daemonEnvVar)
	return
}

////////////////////////////////////////////////////////////////////////
// Command
////////////////////////////////////////////////////////////////////////

// Start the file system daemon and wait for it to mount successfully.
func startDaemon(ctx context.Context, args []string) (err error) {
	// Prompt the user for their password here instead of in the daemon.
	password := getPassword()

	// Find our executable.
	path, err := osext.Executable()
	if err != nil {
		err = fmt.Errorf("osext.Executable: %v", err)
		return
	}

	// Re-execute as the daemon, forwarding status output to stderr.
	err = daemonize.Run(
		path,
		append([]string{"mount"}, args...),
		[]string{
			fmt.Sprintf("%s=%s", passwordEnvVar, password),
			fmt.Sprintf("%s=", daemonEnvVar),
		},
		os.Stderr)

	if err != nil {
		err = fmt.Errorf("daemonize.Run: %v", err)
		return
	}

	return
}

func mount(
	ctx context.Context,
	args []string,
	logger *log.Logger) (mfs *fuse.MountedFileSystem, err error) {
	// Enable invariant checking for the file system.
	syncutil.EnableInvariantChecking()

	// Check usage.
	if len(args) < 1 || len(args) > 2 {
		err = fmt.Errorf("Usage: %s mount_point [score]", os.Args[0])
		return
	}

	mountPoint := args[0]

	// Grab dependencies.
	blobStore := getBlobStore(ctx)

	var hexScore string
	if len(args) > 1 {
		hexScore = args[1]
	}

	// Figure out which score to mount.
	var score blob.Score
	if hexScore != "" {
		score, err = blob.ParseHexScore(hexScore)
		if err != nil {
			err = fmt.Errorf("ParseHexScore(%q): %v", hexScore, err)
			return
		}
	} else {
		r := getRegistry(ctx)

		// List jobs.
		var jobs []registry.CompletedJob
		jobs, err = r.ListBackups(ctx)
		if err != nil {
			err = fmt.Errorf("ListBackups: %v", err)
			return
		}

		if len(jobs) == 0 {
			err = errors.New("No completed jobs found.")
			return
		}

		// Find the job with the newest start time.
		j := jobs[0]
		for _, candidate := range jobs {
			if j.StartTime.Before(candidate.StartTime) {
				j = candidate
			}
		}

		score = j.Score
	}

	logger.Printf("Mounting score %s.", score.Hex())

	// Choose permission settings.
	uid, gid, err := currentUser()
	if err != nil {
		err = fmt.Errorf("currentUser: %v", err)
		return
	}

	// Create the file system.
	fs, err := comebackfs.NewFileSystem(uid, gid, score, blobStore)
	if err != nil {
		err = fmt.Errorf("NewFileSystem: %v", err)
		return
	}

	// Mount it.
	cfg := &fuse.MountConfig{
		FSName:      fmt.Sprintf("comeback-%s", score.Hex()),
		ReadOnly:    true,
		ErrorLogger: log.New(os.Stderr, "fuse: ", log.Flags()),

		// Everything is immutable, so let the kernel cache to its heart's content.
		EnableVnodeCaching: true,
	}

	if *fDebugFuse {
		cfg.DebugLogger = log.New(os.Stderr, "debug_fuse: ", log.Flags())
	}

	mfs, err = fuse.Mount(
		mountPoint,
		fuseutil.NewFileSystemServer(fs),
		cfg)

	if err != nil {
		err = fmt.Errorf("fuse.Mount: %v", err)
		return
	}

	return
}

func runMount(ctx context.Context, args []string) (err error) {
	// If we are not the file system daemon, we should start it and wait for it
	// to mount successfully, then do no more.
	if !isDaemon() {
		err = startDaemon(ctx, args)
		if err != nil {
			err = fmt.Errorf("startDaemon: %v", err)
			return
		}

		return
	}

	// Otherwise, attempt to mount the file system. Communicate status to the
	// program waiting on us to finish mounting.
	logger := log.New(daemonize.StatusWriter, "", 0)

	mfs, err := mount(ctx, args, logger)
	if err != nil {
		err = fmt.Errorf("mount: %v", err)
		daemonize.SignalOutcome(err)
		return
	}

	logger.Println("File system successfully mounted.")
	daemonize.SignalOutcome(nil)

	// Wait for unmount.
	err = mfs.Join(ctx)
	if err != nil {
		err = fmt.Errorf("Join: %v", err)
		return
	}

	return
}

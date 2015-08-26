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
	"os/signal"
	"os/user"
	"strconv"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/comebackfs"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/fuse"
	"github.com/jacobsa/fuse/fuseutil"
	"github.com/jacobsa/syncutil"
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

func registerSIGINTHandler(mountPoint string) {
	// Register for SIGINT.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	// Start a goroutine that will unmount when the signal is received.
	go func() {
		for {
			<-signalChan
			log.Println("Received SIGINT, attempting to unmount...")

			err := fuse.Unmount(mountPoint)
			if err != nil {
				log.Printf("Failed to unmount in response to SIGINT: %v", err)
			} else {
				log.Printf("Successfully unmounted in response to SIGINT.")
				return
			}
		}
	}()
}

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

////////////////////////////////////////////////////////////////////////
// Command
////////////////////////////////////////////////////////////////////////

func runMount(ctx context.Context, args []string) (err error) {
	// Enable invariant checking for the file system.
	syncutil.EnableInvariantChecking()

	// Grab dependencies.
	bucket := getBucket(ctx)
	crypter := getCrypter()

	// Check usage.
	if len(args) < 1 || len(args) > 2 {
		err = fmt.Errorf("Usage: %s mount_point [score]", os.Args[0])
		return
	}

	mountPoint := args[0]

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
		r := getRegistry()

		// List jobs.
		var jobs []registry.CompletedJob
		jobs, err = r.ListBackups()
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

	log.Printf("Mounting score %s.", score.Hex())

	// Create the blob store.
	blobStore, err := wiring.MakeBlobStore(
		bucket,
		crypter,
		util.NewStringSet())

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

	mfs, err := fuse.Mount(
		mountPoint,
		fuseutil.NewFileSystemServer(fs),
		cfg)

	if err != nil {
		err = fmt.Errorf("Mount: %v", err)
		return
	}

	log.Println("File system mounted.")

	// Watch for SIGINT.
	registerSIGINTHandler(mountPoint)

	// Wait for unmount.
	err = mfs.Join(ctx)
	if err != nil {
		err = fmt.Errorf("Join: %v", err)
		return
	}

	log.Println("Exiting successfully.")
	return
}

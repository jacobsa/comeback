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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/config"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/fs"
	s3_kv "github.com/jacobsa/comeback/kv/s3"
	"github.com/jacobsa/comeback/registry"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/sys"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
)

var g_configFile = flag.String("config", "", "Path to config file.")
var g_jobIdStr = flag.String("job_id", "", "The job ID to restore.")
var g_target = flag.String("target", "", "The target directory.")

var g_blobStore blob.Store

func fromHexHash(h string) (blob.Score, error) {
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("Invalid hex string: %s", h)
	}

	return blob.Score(b), nil
}

func chooseUserId(uid sys.UserId, username *string) (sys.UserId, error) {
	// If there is no symbolic username, just return the UID.
	if username == nil {
		return uid, nil
	}

	// Create a user registry.
	registry, err := sys.NewUserRegistry()
	if err != nil {
		return 0, fmt.Errorf("Creating user registry: %v", err)
	}

	// Attempt to look up the username. If it's not found, return the UID.
	betterUid, err := registry.FindByName(*username)

	if _, ok := err.(sys.NotFoundError); ok {
		return uid, nil
	} else if err != nil {
		return 0, fmt.Errorf("Looking up user: %v", err)
	}

	return betterUid, nil
}

func chooseGroupId(gid sys.GroupId, groupname *string) (sys.GroupId, error) {
	// If there is no symbolic groupname, just return the GID.
	if groupname == nil {
		return gid, nil
	}

	// Create a group registry.
	registry, err := sys.NewGroupRegistry()
	if err != nil {
		return 0, fmt.Errorf("Creating group registry: %v", err)
	}

	// Attempt to look up the groupname. If it's not found, return the GID.
	betterGid, err := registry.FindByName(*groupname)

	if _, ok := err.(sys.NotFoundError); ok {
		return gid, nil
	} else if err != nil {
		return 0, fmt.Errorf("Looking up group: %v", err)
	}

	return betterGid, nil
}

// Restore the file whose contents are described by the referenced blobs to the
// supplied target, whose parent must already exist.
func restoreFile(target string, scores []blob.Score) error {
	// Open the file.
	//
	// TODO(jacobsa): Fix permissions race condition here, since we create the
	// file with 0666.
	f, err := os.Create(target)
	defer f.Close()

	if err != nil {
		return fmt.Errorf("Create: %v", err)
	}

	// Process each blob.
	for _, score := range scores {
		// Load the blob.
		blob, err := g_blobStore.Load(score)
		if err != nil {
			return fmt.Errorf("Loading blob: %v", err)
		}

		// Write out its contents.
		_, err = io.Copy(f, bytes.NewReader(blob))
		if err != nil {
			return fmt.Errorf("Copy: %v", err)
		}
	}

	return nil
}

// Restore the directory whose contents are described by the referenced blob to
// the supplied target, which must already exist.
func restoreDir(
	basePath, relPath string,
	score blob.Score,
	fileSystem fs.FileSystem) error {
	// Load the appropriate blob.
	blob, err := g_blobStore.Load(score)
	if err != nil {
		return fmt.Errorf("Loading blob: %v", err)
	}

	// Parse its contents.
	entries, err := repr.Unmarshal(blob)
	if err != nil {
		return fmt.Errorf("Parsing blob: %v", err)
	}

	// Deal with each entry.
	for _, entry := range entries {
		entryRelPath := path.Join(relPath, entry.Name)
		entryFullPath := path.Join(basePath, entryRelPath)

		// Switch on type.
		switch entry.Type {
		case fs.TypeFile:
			// Is this a hard link to another file?
			if entry.HardLinkTarget != nil {
				// Create the hard link.
				targetFullPath := path.Join(basePath, *entry.HardLinkTarget)
				if err := os.Link(targetFullPath, entryFullPath); err != nil {
					return fmt.Errorf("os.Link: %v", err)
				}
			} else {
				// Create the file using its blobs.
				if err := restoreFile(entryFullPath, entry.Scores); err != nil {
					return fmt.Errorf("restoreFile: %v", err)
				}
			}

		case fs.TypeDirectory:
			if len(entry.Scores) != 1 {
				return fmt.Errorf("Wrong number of scores: %v", entry)
			}

			if err = os.Mkdir(entryFullPath, 0700); err != nil {
				return fmt.Errorf("Mkdir: %v", err)
			}

			err = restoreDir(basePath, entryRelPath, entry.Scores[0], fileSystem)
			if err != nil {
				return fmt.Errorf("restoreDir: %v", err)
			}

		case fs.TypeSymlink:
			err = os.Symlink(entry.Target, entryFullPath)
			if err != nil {
				return fmt.Errorf("Symlink: %v", err)
			}

		case fs.TypeNamedPipe:
			err = fileSystem.CreateNamedPipe(entryFullPath, entry.Permissions)
			if err != nil {
				return fmt.Errorf("CreateNamedPipe: %v", err)
			}

		case fs.TypeBlockDevice:
			err = fileSystem.CreateBlockDevice(
				entryFullPath,
				entry.Permissions,
				entry.DeviceNumber)

			if err != nil {
				return fmt.Errorf("CreateBlockDevice: %v", err)
			}

		case fs.TypeCharDevice:
			err = fileSystem.CreateCharDevice(
				entryFullPath,
				entry.Permissions,
				entry.DeviceNumber)

			if err != nil {
				return fmt.Errorf("CreateCharDevice: %v", err)
			}

		default:
			return fmt.Errorf("Don't know how to deal with entry: %v", entry)
		}

		// Fix ownership.
		uid, err := chooseUserId(entry.Uid, entry.Username)
		if err != nil {
			return fmt.Errorf("chooseUserId: %v", err)
		}

		gid, err := chooseGroupId(entry.Gid, entry.Groupname)
		if err != nil {
			return fmt.Errorf("chooseGroupId: %v", err)
		}

		if err = os.Lchown(entryFullPath, int(uid), int(gid)); err != nil {
			return fmt.Errorf("Chown: %v", err)
		}

		// Fix permissions, but not on devices (otherwise we get resource busy
		// errors).
		if entry.Type != fs.TypeBlockDevice && entry.Type != fs.TypeCharDevice {
			err := fileSystem.SetPermissions(entryFullPath, entry.Permissions)
			if err != nil {
				return fmt.Errorf("SetPermissions(%s): %v", entryFullPath, err)
			}
		}

		// Fix modification time, but not on devices (otherwise we get resource
		// busy errors).
		if entry.Type != fs.TypeBlockDevice && entry.Type != fs.TypeCharDevice {
			err = fileSystem.SetModTime(entryFullPath, entry.MTime)
			if err != nil {
				return fmt.Errorf("SetModTime(%s): %v", entryFullPath, err)
			}
		}
	}

	return nil
}

func main() {
	var err error
	flag.Parse()

	// Parse the job ID.
	if len(*g_jobIdStr) != 16 {
		fmt.Println("You must set -job_id.")
		os.Exit(1)
	}

	jobId, err := strconv.ParseUint(*g_jobIdStr, 16, 64)
	if err != nil {
		fmt.Println("Invalid job ID:", *g_jobIdStr)
		os.Exit(1)
	}

	// Check the target.
	if *g_target == "" {
		fmt.Println("You must set -target.")
		os.Exit(1)
	}

	// Attempt to read the user's config data.
	if *g_configFile == "" {
		fmt.Println("You must set -config.")
		os.Exit(1)
	}

	configData, err := ioutil.ReadFile(*g_configFile)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		os.Exit(1)
	}

	// Parse the config file.
	cfg, err := config.Parse(configData)
	if err != nil {
		fmt.Println("Parsing config file:", err)
		os.Exit(1)
	}

	// Read in the AWS access key secret.
	cfg.AccessKey.Secret = readPassword("Enter AWS access key secret: ")
	if len(cfg.AccessKey.Secret) == 0 {
		log.Fatalf("You must enter an access key secret.\n")
	}

	// Validate the config file.
	if err := config.Validate(cfg); err != nil {
		fmt.Printf("Config file invalid: %v\n", err)
		os.Exit(1)
	}

	// Create a user registry.
	userRegistry, err := sys.NewUserRegistry()
	if err != nil {
		log.Fatalf("Creating user registry: %v", err)
	}

	// Create a group registry.
	groupRegistry, err := sys.NewGroupRegistry()
	if err != nil {
		log.Fatalf("Creating group registry: %v", err)
	}

	// Create a file system.
	fileSystem, err := fs.NewFileSystem(userRegistry, groupRegistry)
	if err != nil {
		log.Fatalf("Creating file system: %v", err)
	}

	// Open a connection to SimpleDB.
	db, err := sdb.NewSimpleDB(cfg.SdbRegion, cfg.AccessKey)
	if err != nil {
		log.Fatalf("Creating SimpleDB: %v", err)
	}

	// Open the appropriate domain.
	domain, err := db.OpenDomain(cfg.SdbDomain)
	if err != nil {
		log.Fatalf("OpenDomain: %v", err)
	}

	// Open a connection to S3.
	bucket, err := s3.OpenBucket(cfg.S3Bucket, cfg.S3Region, cfg.AccessKey)
	if err != nil {
		log.Fatalf("Creating bucket: %v", err)
	}

	// Read in the password.
	password := readPassword("Enter crypto password: ")
	if len(password) == 0 {
		log.Fatalf("You must enter a password.")
	}

	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32 // Minimum key length for AES-SIV
	keyDeriver := crypto.NewPbkdf2KeyDeriver(pbkdf2Iters, keyLen, sha256.New)

	// Create the backup registry.
	reg, crypter, err := registry.NewRegistry(domain, password, keyDeriver)
	if err != nil {
		log.Fatalf("Creating registry: %v", err)
	}

	// Create the kv store.
	kvStore, err := s3_kv.NewS3KvStore(bucket)
	if err != nil {
		log.Fatalf("Creating kv store: %v", err)
	}

	// Create the blob store.
	g_blobStore = blob.NewKvBasedBlobStore(kvStore)
	g_blobStore = blob.NewCheckingStore(g_blobStore)
	g_blobStore = blob.NewEncryptingStore(crypter, g_blobStore)

	// Find the requested job.
	job, err := reg.FindBackup(jobId)
	if err != nil {
		log.Fatalln("FindBackup:", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(*g_target)
	if err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}

	// Create the target.
	err = os.Mkdir(*g_target, 0755)
	if err != nil {
		log.Fatalf("Mkdir: %v", err)
	}

	// Attempt a restore.
	err = restoreDir(*g_target, "", job.Score, fileSystem)
	if err != nil {
		log.Fatalf("Restoring: %v", err)
	}
}

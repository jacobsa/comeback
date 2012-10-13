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
	"code.google.com/p/go.crypto/pbkdf2"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/config"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/fs"
	s3_kv "github.com/jacobsa/comeback/kv/s3"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/sys"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"
)

var g_configFile = flag.String("config", "", "Path to config file.")
var g_jobIdStr = flag.String("job_id", "", "The job ID to restore.")
var g_target = flag.String("target", "", "The target directory.")

var g_blobStore blob.Store

// Keep this consistent with the item name used in the backup package.
const markerItemName = "comeback_marker"
const saltAttributeName = "password_salt"

func getSalt(domain sdb.Domain) (salt []byte, err error) {
	// Call the domain.
	attrs, err := domain.GetAttributes(
		markerItemName,
		false, // No need to ask for a consistent read
		[]string{saltAttributeName},
	)

	if err != nil {
		err = fmt.Errorf("GetAttributes: %v", err)
		return
	}

	if len(attrs) == 0 {
		err = fmt.Errorf("Couldn't find salt in the supplied domain.")
		return
	}

	if attrs[0].Name != saltAttributeName {
		panic(fmt.Errorf("Unexpected attribute: %v", attrs[0]))
	}

	// Base64-decode the salt.
	salt, err = base64.StdEncoding.DecodeString(attrs[0].Value)
	if err != nil {
		err = fmt.Errorf("base64.DecodeString(%s): %v", attrs[0].Value, err)
		return
	}

	return
}

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

// Set the modification time for the supplied path without following symlinks
// (as syscall.Chtimes and therefore os.Chtimes do).
//
// c.f. http://stackoverflow.com/questions/10608724/set-modification-date-on-symbolic-link-in-cocoa
func setModTime(path string, mtime time.Time) error {
	// Open the file without following symlinks. Use O_NONBLOCK to allow opening
	// of named pipes without a writer.
	fd, err := syscall.Open(path, syscall.O_NONBLOCK|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call futimes.
	var utimes [2]syscall.Timeval
	atime := time.Now()
	atime_ns := atime.Unix()*1e9 + int64(atime.Nanosecond())
	mtime_ns := mtime.Unix()*1e9 + int64(mtime.Nanosecond())
	utimes[0] = syscall.NsecToTimeval(atime_ns)
	utimes[1] = syscall.NsecToTimeval(mtime_ns)

	err = syscall.Futimes(fd, utimes[0:])
	if err != nil {
		return err
	}

	return nil
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

// Like os.Chmod, but don't follow symlinks.
func setPermissions(path string, permissions os.FileMode) error {
	mode := syscallPermissions(permissions)

	// Open the file without following symlinks. Use O_NONBLOCK to allow opening
	// of named pipes without a writer.
	fd, err := syscall.Open(path, syscall.O_NONBLOCK|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call fchmod.
	err = syscall.Fchmod(fd, mode)
	if err != nil {
		return err
	}

	return nil
}

// Restore the directory whose contents are described by the referenced blob to
// the supplied target, which must already exist.
func restoreDir(basePath, relPath string, score blob.Score) error {
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

			if err = restoreDir(basePath, entryRelPath, entry.Scores[0]); err != nil {
				return fmt.Errorf("restoreDir: %v", err)
			}

		case fs.TypeSymlink:
			err = os.Symlink(entry.Target, entryFullPath)
			if err != nil {
				return fmt.Errorf("Symlink: %v", err)
			}

		case fs.TypeNamedPipe:
			err = makeNamedPipe(entryFullPath, entry.Permissions)
			if err != nil {
				return fmt.Errorf("makeNamedPipe: %v", err)
			}

		case fs.TypeBlockDevice:
			err = makeBlockDevice(entryFullPath, entry.Permissions, entry.DeviceNumber)
			if err != nil {
				return fmt.Errorf("makeBlockDevice: %v", err)
			}

		case fs.TypeCharDevice:
			err = makeCharDevice(entryFullPath, entry.Permissions, entry.DeviceNumber)
			if err != nil {
				return fmt.Errorf("makeCharDevice: %v", err)
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
			if err := setPermissions(entryFullPath, entry.Permissions); err != nil {
				return fmt.Errorf("setPermissions(%s): %v", entryFullPath, err)
			}
		}

		// Fix modification time, but not on devices (otherwise we get resource
		// busy errors).
		if entry.Type != fs.TypeBlockDevice && entry.Type != fs.TypeCharDevice {
			if err = setModTime(entryFullPath, entry.MTime); err != nil {
				return fmt.Errorf("setModTime(%s): %v", entryFullPath, err)
			}
		}
	}

	return nil
}

func syscallPermissions(permissions os.FileMode) (o uint32) {
	// Include r/w/x permission bits.
	o = uint32(permissions & os.ModePerm)

	// Also include setuid/setgid/sticky bits.
	if permissions&os.ModeSetuid != 0 {
		o |= syscall.S_ISUID
	}

	if permissions&os.ModeSetgid != 0 {
		o |= syscall.S_ISGID
	}

	if permissions&os.ModeSticky != 0 {
		o |= syscall.S_ISVTX
	}

	return
}

// Create a named pipe at the supplied path.
func makeNamedPipe(path string, permissions os.FileMode) error {
	return syscall.Mkfifo(path, syscallPermissions(permissions))
}

// Create a block device at the supplied path.
func makeBlockDevice(path string, permissions os.FileMode, dev int32) error {
	mode := syscallPermissions(permissions) | syscall.S_IFBLK
	if err := syscall.Mknod(path, mode, int(dev)); err != nil {
		return fmt.Errorf("syscall.Mknod: %v", err)
	}

	return nil
}

// Create a character device at the supplied path.
func makeCharDevice(path string, permissions os.FileMode, dev int32) error {
	mode := syscallPermissions(permissions) | syscall.S_IFCHR
	if err := syscall.Mknod(path, mode, int(dev)); err != nil {
		return fmt.Errorf("syscall.Mknod: %v", err)
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
	password := []byte(readPassword("Enter crypto password: "))
	if len(password) == 0 {
		log.Fatalf("You must enter a password.")
	}

	// Load the salt.
	salt, err := getSalt(domain)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32 // Minimum key length for AES-SIV
	cryptoKey := pbkdf2.Key(password, salt, pbkdf2Iters, keyLen, sha256.New)

	// Create the crypter.
	crypter, err := crypto.NewCrypter(cryptoKey)
	if err != nil {
		log.Fatalf("Creating crypter: %v", err)
	}

	// Create the backup registry.
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	registry, err := backup.NewRegistry(domain, crypter, randSrc)
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

	// Find the requested job.
	job, err := registry.FindBackup(jobId)
	if err != nil {
		log.Fatalln("FindBackup:", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(*g_target)
	if err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}

	// Create the target.
	err = os.Mkdir("/tmp/restore_target", 0755)
	if err != nil {
		log.Fatalf("Mkdir: %v", err)
	}

	// Attempt a restore.
	err = restoreDir(*g_target, "", job.Score)
	if err != nil {
		log.Fatalf("Restoring: %v", err)
	}
}

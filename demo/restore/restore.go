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
	"encoding/hex"
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/disk"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/sys"
	"io"
	"log"
	"os"
	"path"
	"syscall"
	"time"
)

const (
	gTarget = "/tmp/restore_target"
)

var blobStore blob.Store

type score struct {
	hash []byte
}

func (s *score) Sha1Hash() []byte {
	return s.hash
}

func fromHexHash(h string) (blob.Score, error) {
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("Invalid hex string: %s", h)
	}

	return &score{b}, nil
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
	f, err := os.Create(target)
	defer f.Close()

	if err != nil {
		return fmt.Errorf("Create: %v", err)
	}

	// Process each blob.
	for _, score := range scores {
		// Load the blob.
		blob, err := blobStore.Load(score)
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

	// Open the file without following symlinks.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_SYMLINK, 0)
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
func restoreDir(target string, score blob.Score) error {
	// Load the appropriate blob.
	blob, err := blobStore.Load(score)
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
		entryPath := path.Join(target, entry.Name)

		// Switch on type.
		switch entry.Type {
		case fs.TypeFile:
			if err := restoreFile(entryPath, entry.Scores); err != nil {
				return err
			}

			if err := setPermissions(entryPath, entry.Permissions); err != nil {
				return err
			}

		case fs.TypeDirectory:
			if len(entry.Scores) != 1 {
				return fmt.Errorf("Wrong number of scores: %v", entry)
			}

			if err = os.Mkdir(entryPath, 0700); err != nil {
				return err
			}

			if err = restoreDir(entryPath, entry.Scores[0]); err != nil {
				return err
			}

		case fs.TypeSymlink:
			err = os.Symlink(entry.Target, entryPath)
			if err != nil {
				return err
			}

		case fs.TypeNamedPipe:
			err = makeNamedPipe(entryPath, entry.Permissions)
			if err != nil {
				return err
			}

		case fs.TypeBlockDevice:
			err = makeBlockDevice(entryPath, entry.Permissions, entry.Device)
			if err != nil {
				return err
			}

		case fs.TypeCharDevice:
			err = makeCharDevice(entryPath, entry.Permissions, entry.Device)
			if err != nil {
				return err
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

		if err = os.Lchown(entryPath, int(uid), int(gid)); err != nil {
			return fmt.Errorf("Chown: %v", err)
		}

		// Fix modification time, but not on devices (otherwise we get resource
		// busy errors).
		if entry.Type != fs.TypeBlockDevice && entry.Type != fs.TypeCharDevice {
			if err = setModTime(entryPath, entry.MTime); err != nil {
				return fmt.Errorf("setModTime(%s): %v", entryPath, err)
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

	// Create the blob store.
	blobStore, err = disk.NewBlobStore("/tmp/blobs")
	if err != nil {
		log.Fatalf("Creating store: %v", err)
	}

	// Parse the score.
	score, err := fromHexHash("228a6254c7585525744192b51f099833fca8c654")
	if err != nil {
		log.Fatalf("Parsing score: %v", err)
	}

	// Make sure the target doesn't exist.
	err = os.RemoveAll(gTarget)
	if err != nil {
		log.Fatalf("RemoveAll: %v", err)
	}

	// Create the target.
	err = os.Mkdir("/tmp/restore_target", 0755)
	if err != nil {
		log.Fatalf("Mkdir: %v", err)
	}

	// Attempt a restore.
	err = restoreDir(gTarget, score)
	if err != nil {
		log.Fatalf("Restoring: %v", err)
	}
}

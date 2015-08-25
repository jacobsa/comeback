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

package fs

import (
	"fmt"
	"os"
	"syscall"

	"github.com/jacobsa/comeback/internal/sys"
)

const (
	// A permissions bit mask that matches chmod(2)'s notion of permissions.
	permissionBits os.FileMode = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
)

var gUserRegistry = sys.NewUserRegistry()
var gGroupRegistry = sys.NewGroupRegistry()

// Convert the result of os.Lstat or os.Stat to a directory entry.
// symlinkTarget should be empty if this is not a symlink.
func ConvertFileInfo(
	fi os.FileInfo,
	symlinkTarget string) (entry *DirectoryEntry, err error) {
	entry, err = convertFileInfo(fi, symlinkTarget, gUserRegistry, gGroupRegistry)
	return
}

// Like ConvertFileInfo, but allows injecting registries.
func convertFileInfo(
	fi os.FileInfo,
	symlinkTarget string,
	userRegistry sys.UserRegistry,
	groupRegistry sys.GroupRegistry) (entry *DirectoryEntry, err error) {
	// Grab system-specific info.
	statT, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("Unexpected sys value: %v", fi.Sys())
	}

	if statT.Size < 0 {
		panic(fmt.Sprintf("Unexpected size: %d", statT.Size))
	}

	// Create the basic entry.
	entry = &DirectoryEntry{
		Name:             fi.Name(),
		Permissions:      fi.Mode() & permissionBits,
		Uid:              sys.UserId(statT.Uid),
		Gid:              sys.GroupId(statT.Gid),
		MTime:            fi.ModTime(),
		Size:             uint64(statT.Size),
		ContainingDevice: statT.Dev,
		Inode:            statT.Ino,
		Target:           symlinkTarget,
	}

	// Attempt to look up user info.
	username, err := userRegistry.FindById(entry.Uid)

	if _, ok := err.(sys.NotFoundError); ok {
		err = nil
	} else if err != nil {
		return nil, fmt.Errorf("Looking up user: %v", err)
	} else {
		entry.Username = &username
	}

	// Attempt to look up group info.
	groupname, err := groupRegistry.FindById(entry.Gid)

	if _, ok := err.(sys.NotFoundError); ok {
		err = nil
	} else if err != nil {
		return nil, fmt.Errorf("Looking up group: %v", err)
	} else {
		entry.Groupname = &groupname
	}

	// Convert the type.
	typeBits := fi.Mode() & (os.ModeType | os.ModeCharDevice)
	switch typeBits {
	case 0:
		entry.Type = TypeFile
	case os.ModeDir:
		entry.Type = TypeDirectory
	case os.ModeSymlink:
		entry.Type = TypeSymlink
	case os.ModeDevice:
		entry.Type = TypeBlockDevice
		entry.DeviceNumber = statT.Rdev
	case os.ModeDevice | os.ModeCharDevice:
		entry.Type = TypeCharDevice
		entry.DeviceNumber = statT.Rdev
	case os.ModeNamedPipe:
		entry.Type = TypeNamedPipe
	case os.ModeSocket:
		entry.Type = TypeSocket
	default:
		return entry, fmt.Errorf("Unhandled mode: %v %u", fi.Mode(), fi.Mode())
	}

	return entry, nil
}

func (fs *fileSystem) Stat(path string) (entry DirectoryEntry, err error) {
	// Call lstat.
	fi, err := os.Lstat(path)
	if err != nil {
		err = fmt.Errorf("Lstat: %v", err)
		return
	}

	// Read the symlink target, if any.
	var symlinkTarget string
	if fi.Mode()&os.ModeSymlink != 0 {
		symlinkTarget, err = os.Readlink(path)
		if err != nil {
			err = fmt.Errorf("Readlink: %v", err)
			return
		}
	}

	// Convert to an entry.
	entryPtr, err := convertFileInfo(
		fi,
		symlinkTarget,
		fs.userRegistry,
		fs.groupRegistry)

	if err != nil {
		err = fmt.Errorf("convertFileInfo: %v", err)
		return
	}

	entry = *entryPtr
	return
}

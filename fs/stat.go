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
	"github.com/jacobsa/comeback/sys"
	"os"
	"syscall"
)

const (
	// A permissions bit mask that matches chmod(2)'s notion of permissions.
	permissionBits os.FileMode = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
)

func (fs *fileSystem) convertFileInfo(fi os.FileInfo) (entry *DirectoryEntry, err error) {
	// Grab system-specific info.
	statT, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("Unexpected sys value: %v", fi.Sys())
	}

	// Create the basic entry.
	entry = &DirectoryEntry{
		Permissions: fi.Mode() & permissionBits,
		Name:        fi.Name(),
		MTime:       fi.ModTime(),
		Uid:         sys.UserId(statT.Uid),
		Gid:         sys.GroupId(statT.Gid),
	}

	// Attempt to look up user info.
	username, err := fs.userRegistry.FindById(entry.Uid)

	if _, ok := err.(sys.NotFoundError); ok {
		err = nil
	} else if err != nil {
		return nil, fmt.Errorf("Looking up user: %v", err)
	} else {
		entry.Username = &username
	}

	// Attempt to look up group info.
	groupname, err := fs.groupRegistry.FindById(entry.Gid)

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
		entry.ContainingDevice = statT.Dev
		entry.Inode = statT.Ino

		if statT.Size < 0 {
			panic(fmt.Sprintf("Unexpected size: %d", statT.Size))
		}

		entry.Size = uint64(statT.Size)

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
	default:
		return entry, fmt.Errorf("Unhandled mode: %v %u", fi.Mode(), fi.Mode())
	}

	return entry, nil
}

func (fs *fileSystem) Stat(path string) (entry DirectoryEntry, err error) {
	// Call stat.
	var fi os.FileInfo
	if fi, err = os.Lstat(path); err != nil {
		return
	}

	// Convert to an entry.
	var entryPtr *DirectoryEntry
	if entryPtr, err = fs.convertFileInfo(fi); err != nil {
		return
	}

	entry = *entryPtr

	// Convert symlinks.
	if entry.Type == TypeSymlink {
		if entry.Target, err = os.Readlink(path); err != nil {
			return
		}
	}

	return
}

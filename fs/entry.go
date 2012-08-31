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

// Package fs contains file system related functions and types.
package fs

import (
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/sys"
	"os"
	"time"
)

type EntryType uint32

const (
	TypeFile EntryType = iota
	TypeDirectory
	TypeSymlink
	TypeBlockDevice
	TypeCharDevice
	TypeNamedPipe
)

// DirectoryEntry gives enough information to reconstruct a single entry within
// a backed up directory.
type DirectoryEntry struct {
	Type EntryType

	// The name of this entry within its directory.
	Name string

	// The permissions for this entry, including the {setuid,setgid,sticky} bits.
	// That is, the things that chmod(2) cares about. This does *not* include
	// type information such as os.ModeDevice or options such as os.ModeAppend.
	Permissions os.FileMode

	// The owning user's UID, and their username if known.
	Uid      sys.UserId
	Username *string

	// The owning group's GID, and its groupname if known.
	Gid       sys.GroupId
	Groupname *string

	// The modification time of this entry.
	MTime time.Time

	// The containing device's device number, and the inode on the device. These
	// are defined only for regular files.
	ContainingDevice int32
	Inode uint64

	// The scores of zero or more blobs that make up a regular file's contents,
	// to be concatenated in order. For directories, this is exactly one blob
	// whose contents can be processed using repr.Unmarshal.
	//
	// Scores are present only if HardLinkTarget is not present.
	Scores []blob.Score

	// If this regular file belongs to a backup containing another regular file
	// to which it is hard linked, this is the target of the hard link relative
	// to the root of the backup.
	HardLinkTarget *string

	// The target, if this is a symlink.
	Target string

	// The device number, for devices.
	DeviceNumber int32
}

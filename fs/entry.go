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

	// The modification time of this entry.
	MTime time.Time

	// The scores of zero or more blobs that make up a regular file's contents,
	// to be concatenated in order. For directories, this is exactly one blob
	// whose contents can be processed using repr.Unmarshal.
	Scores []blob.Score

	// The target, if this is a symlink.
	Target string

	// The device number, for devices.
	Device int32
}

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
	"time"
)

type EntryType uint32

const (
	TypeFile EntryType = iota
	TypeDirectory
	TypeSymlink
)

// DirectoryEntry gives enough information to reconstruct a single entry within
// a backed up directory.
type DirectoryEntry struct {
	Type EntryType

	// The permission bits for this entry.
	Permissions uint32

	// The name of this entry within its directory.
	Name []byte

	// The modification time of this entry.
	MTime time.Time

	// The scores of zero or more blobs that make up a regular file's contents,
	// to be concatenated in order.
	Scores []blob.Score
}

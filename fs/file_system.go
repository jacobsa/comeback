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
	"github.com/jacobsa/comeback/sys"
	"io"
	"os"
	"time"
)

// FileSystem represents operations performed on a real file system, but is an
// interface for mockability.
type FileSystem interface {
	// Read the contents of the directory named by the supplied path, returning
	// an array of directory entries sorted by name. The entries will contain no
	// scores.
	ReadDir(path string) (entries []*DirectoryEntry, err error)

	// Open the file named by the supplied path for reading.
	OpenForReading(path string) (r io.ReadCloser, err error)

	// Write out the supplied data to the supplied path, truncating if the file
	// already exists and creating with the supplied permissions otherwise.
	WriteFile(path string, data []byte, permissions os.FileMode) error

	// Set the modification time for the supplied path, not following symlinks.
	SetModTime(path string, mtime time.Time) error

	// Set permissions for the supplied path, not following symlinks.
	SetPermissions(path string, permissions os.FileMode) error

	// Create a file system object of various types.
	CreateNamedPipe(path string, permissions os.FileMode) error
	CreateBlockDevice(path string, permissions os.FileMode, devNum int32) error
	CreateCharDevice(path string, permissions os.FileMode, devNum int32) error
}

// Return a FileSystem that uses the real file system, along with the supplied
// registries.
func NewFileSystem(
	userRegistry sys.UserRegistry,
	groupRegistry sys.GroupRegistry) (fs FileSystem, err error) {
	return &fileSystem{userRegistry, groupRegistry}, nil
}

type fileSystem struct {
	userRegistry  sys.UserRegistry
	groupRegistry sys.GroupRegistry
}

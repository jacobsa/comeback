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
	"io"
	"io/ioutil"
	"os"
)

// FileSystem represents operations performed on a real file system, but is an
// interface for mockability.
type FileSystem interface {
	// Read the contents of the directory named by the supplied path, returning
	// an array of directory entries sorted by name.
	ReadDir(path string) (entries []*DirectoryEntry, err error)

	// Open the file named by the supplied path for reading.
	OpenForReading(path string) (r io.Reader, err error)
}

// Return a FileSystem that uses the read file system.
func NewFileSystem() FileSystem {
	return &fileSystem{}
}

type fileSystem struct {
}

func (f *fileSystem) ReadDir(path string) (entries []*DirectoryEntry, err error) {
	return ioutil.ReadDir(path)
}

func (f *fileSystem) OpenForReading(path string) (r io.Reader, err error) {
	return os.Open(path)
}

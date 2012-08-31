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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

const (
	// A permissions bit mask that matches chmod(2)'s notion of permissions.
	permissionBits os.FileMode = os.ModePerm|os.ModeSetuid|os.ModeSetgid|os.ModeSticky
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
}

// Return a FileSystem that uses the read file system.
func NewFileSystem() FileSystem {
	return &fileSystem{}
}

type fileSystem struct {
}

func convertFileInfo(fi os.FileInfo) (*DirectoryEntry, error) {
	entry := &DirectoryEntry{
		Permissions: fi.Mode() & permissionBits,
		Name:        fi.Name(),
		MTime:       fi.ModTime(),
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
	case os.ModeDevice | os.ModeCharDevice:
		entry.Type = TypeCharDevice
	case os.ModeNamedPipe:
		entry.Type = TypeNamedPipe
	default:
		return entry, fmt.Errorf("Unhandled mode: %v %u", fi.Mode(), fi.Mode())
	}

	return entry, nil
}

func (f *fileSystem) ReadDir(dirpath string) (entries []*DirectoryEntry, err error) {
	// Call ioutil.
	fileInfos, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	// Convert each entry.
	entries = []*DirectoryEntry{}
	for _, fileInfo := range fileInfos {
		entry, err := convertFileInfo(fileInfo)
		if err != nil {
			return nil, err
		}

		// Handle symlinks.
		if entry.Type == TypeSymlink {
			linkPath := path.Join(dirpath, entry.Name)
			if entry.Target, err = os.Readlink(linkPath); err != nil {
				return nil, err
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (f *fileSystem) OpenForReading(path string) (r io.ReadCloser, err error) {
	return os.Open(path)
}

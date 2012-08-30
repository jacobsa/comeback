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

package backup

import (
	"errors"
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr"
	"path"
)

// An object that knows how to save directories to some underlying storage.
type DirectorySaver interface {
	// Recursively save the contents of the supplied directory to the underlying
	// storage, returning the score of a blob representing the directory's
	// listing in a format hat can be recovered with repr.Unmarshal.
	Save(dirpath string) (score blob.Score, err error)
}

// A directory saver that creates a new directory saver for each call to Save.
// This breaks a self-dependency that would be needed to make use of
// NewNonRecursiveDirectorySaver.
type onDemandDirSaver struct {
	createSaver func(wrapped DirectorySaver) DirectorySaver
}

func (s *onDemandDirSaver) Save(dirpath string) (score blob.Score, err error) {
	return s.createSaver(s).Save(dirpath)
}

// Return a directory saver that makes use of the supplied dependencies.
func NewDirectorySaver(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
	fileSaver FileSaver) (DirectorySaver, error) {
	createSaver := func(wrapped DirectorySaver) DirectorySaver {
		saver, err := NewNonRecursiveDirectorySaver(
			blobStore,
			fileSystem,
			fileSaver,
			wrapped)

		if err != nil {
			panic(err)
		}

		return saver
	}

	return &onDemandDirSaver{createSaver}, nil
}

// Equivalent to NewDirectorySaver, but with an injectable wrapped directory
// saver to aid with testability. You should not use this function.
func NewNonRecursiveDirectorySaver(
	store blob.Store,
	fileSystem fs.FileSystem,
	fileSaver FileSaver,
	wrapped DirectorySaver) (DirectorySaver, error) {
	return &dirSaver{
		blobStore:  store,
		fileSystem: fileSystem,
		fileSaver:  fileSaver,
		wrapped:    wrapped,
	}, nil
}

type dirSaver struct {
	blobStore  blob.Store
	fileSystem fs.FileSystem
	fileSaver  FileSaver
	wrapped    DirectorySaver
}

func (s *dirSaver) saveDir(parent string, entry *fs.DirectoryEntry) ([]blob.Score, error) {
	// Recurse.
	score, err := s.wrapped.Save(path.Join(parent, entry.Name))
	if err != nil {
		return nil, err
	}

	return []blob.Score{score}, nil
}

func (s *dirSaver) saveFile(parent string, entry *fs.DirectoryEntry) ([]blob.Score, error) {
	// Open the file.
	f, err := s.fileSystem.OpenForReading(path.Join(parent, entry.Name))
	if err != nil {
		return nil, fmt.Errorf("Opening file: %v", err)
	}
	defer f.Close()

	// Defer to the file saver.
	return s.fileSaver.Save(f)
}

func (s *dirSaver) Save(dirpath string) (score blob.Score, err error) {
	// Grab a listing for the directory.
	entries, err := s.fileSystem.ReadDir(dirpath)
	if err != nil {
		return nil, fmt.Errorf("Listing directory: %v", err)
	}

	// Save the data for each entry.
	for _, entry := range entries {
		// Call the appropriate method based on this entry's type.
		switch entry.Type {
		case fs.TypeFile:
			entry.Scores, err = s.saveFile(dirpath, entry)
		case fs.TypeDirectory:
			entry.Scores, err = s.saveDir(dirpath, entry)
		default:
			err = fmt.Errorf("Unhandled type: %v", entry.Type)
		}

		if err != nil {
			return nil, err
		}
	}

	// Create a serialized version of this information.
	data, err := repr.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("Marshaling: %v", err)
	}

	// Store that serialized version.
	score, err = s.blobStore.Store(data)
	if err != nil {
		err = errors.New("Storing dir blob: " + err.Error())
	}

	return
}

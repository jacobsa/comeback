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
	"log"
	"path"
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
)

// An object that knows how to save directories to some underlying storage.
type DirectorySaver interface {
	// Recursively save the contents of the supplied directory (defined by a base
	// path of a backup and a relative path within the backup) to the underlying
	// storage, returning the score of a blob representing the directory's
	// listing in a format that can be recovered with repr.Unmarshal.
	//
	// Recursively exclude from backup relative paths that match any of the
	// supplied exclusion regexps. This is *not* tested against the initial
	// relative path.
	//
	// Note that the backup is not guaranteed to be durable until Flush is
	// successfully called.
	Save(basePath, relPath string, exclusions []*regexp.Regexp) (blob.Score, error)

	// Flush previous saves to durable storage. Save must not be called again.
	Flush() (err error)
}

// A directory saver that creates a new directory saver for each call to Save.
// This breaks a self-dependency that would be needed to make use of
// NewNonRecursiveDirectorySaver.
type onDemandDirSaver struct {
	createSaver func(wrapped DirectorySaver) DirectorySaver
}

func (s *onDemandDirSaver) Save(
	basePath,
	relPath string,
	exclusions []*regexp.Regexp) (
	score blob.Score,
	err error) {
	return s.createSaver(s).Save(basePath, relPath, exclusions)
}

func (s *onDemandDirSaver) Flush() (err error) {
	err = s.createSaver(s).Flush()
	return
}

// Return a directory saver that makes use of the supplied dependencies.
func NewDirectorySaver(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
	fileSaver FileSaver,
	logger *log.Logger) (DirectorySaver, error) {
	createSaver := func(wrapped DirectorySaver) DirectorySaver {
		saver, err := NewNonRecursiveDirectorySaver(
			blobStore,
			fileSystem,
			fileSaver,
			wrapped,
			logger)

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
	wrapped DirectorySaver,
	logger *log.Logger) (DirectorySaver, error) {
	return &dirSaver{
		blobStore:  store,
		fileSystem: fileSystem,
		fileSaver:  fileSaver,
		wrapped:    wrapped,
		logger:     logger,
	}, nil
}

type dirSaver struct {
	blobStore  blob.Store
	fileSystem fs.FileSystem
	fileSaver  FileSaver
	wrapped    DirectorySaver
	logger     *log.Logger
}

func (s *dirSaver) Flush() (err error) {
	err = s.blobStore.Flush(context.TODO())
	if err != nil {
		err = fmt.Errorf("blobStore.Flush: %v", err)
		return
	}

	return
}

func (s *dirSaver) saveDir(
	basePath string,
	relPath string,
	exclusions []*regexp.Regexp,
	entry *fs.DirectoryEntry) (
	[]blob.Score,
	error) {
	// Recurse.
	score, err := s.wrapped.Save(
		basePath,
		path.Join(relPath, entry.Name),
		exclusions)

	if err != nil {
		return nil, err
	}

	return []blob.Score{score}, nil
}

func (s *dirSaver) saveFile(
	parent string,
	entry *fs.DirectoryEntry) ([]blob.Score, error) {
	// Defer to the file saver.
	return s.fileSaver.Save(path.Join(parent, entry.Name))
}

func shouldExclude(exclusions []*regexp.Regexp, relPath string) bool {
	for _, re := range exclusions {
		if re.MatchString(relPath) {
			return true
		}
	}

	return false
}

func (s *dirSaver) Save(
	basePath,
	relPath string,
	exclusions []*regexp.Regexp) (
	score blob.Score,
	err error) {
	dirpath := path.Join(basePath, relPath)

	// Grab a listing for the directory.
	entries, err := s.fileSystem.ReadDir(dirpath)
	if err != nil {
		err = fmt.Errorf("Listing directory: %v", err)
		return
	}

	// Filter the entries according to the list of exclusions.
	var tmp []*fs.DirectoryEntry
	for _, entry := range entries {
		if !shouldExclude(exclusions, path.Join(relPath, entry.Name)) {
			tmp = append(tmp, entry)
		}
	}

	entries = tmp

	// Save the data for each entry.
	for _, entry := range entries {
		s.logger.Println("Processing:", path.Join(relPath, entry.Name))

		// Call the appropriate method based on this entry's type.
		switch entry.Type {
		case fs.TypeFile:
			entry.Scores, err = s.saveFile(dirpath, entry)

		case fs.TypeDirectory:
			entry.Scores, err = s.saveDir(basePath, relPath, exclusions, entry)

		case fs.TypeSymlink:
		case fs.TypeBlockDevice:
		case fs.TypeCharDevice:
		case fs.TypeNamedPipe:
		default:
			err = fmt.Errorf("Unhandled type: %v", entry.Type)
		}

		if err != nil {
			return
		}
	}

	// Create a serialized version of this information.
	data, err := repr.MarshalDir(entries)
	if err != nil {
		err = fmt.Errorf("Marshaling: %v", err)
		return
	}

	// Store that serialized version.
	score, err = s.blobStore.Store(context.TODO(), data)
	if err != nil {
		err = errors.New("Storing dir blob: " + err.Error())
		return
	}

	return
}
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
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
)

////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////

// An object that knows how to restore previously backed up directories.
type DirectoryRestorer interface {
	// Recursively restore a directory based on the listing named by the supplied
	// score. The first call should set basePath to the target directory and
	// relPath to the empty string.
	RestoreDirectory(score blob.Score, basePath, relPath string) (err error)
}

// Create a directory restorer that uses the supplied objects.
func NewDirectoryRestorer(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
	fileRestorer FileRestorer,
) (restorer DirectoryRestorer, err error) {
	createRestorer := func(wrapped DirectoryRestorer) DirectoryRestorer {
		restorer, err := NewNonRecursiveDirectoryRestorer(
			blobStore,
			fileSystem,
			fileRestorer,
			wrapped,
		)

		if err != nil {
			panic(err)
		}

		return restorer
	}

	return &onDemandDirRestorer{createRestorer}, nil
}

////////////////////////////////////////////////////////////////////////
// Implementation details
////////////////////////////////////////////////////////////////////////

// A directory restorer that creates a new directory restorer for each call.
// This breaks a self-dependency that would be needed to make use of
// NewNonRecursiveDirectoryRestorer.
type onDemandDirRestorer struct {
	createRestorer func(wrapped DirectoryRestorer) DirectoryRestorer
}

func (r *onDemandDirRestorer) RestoreDirectory(
	score blob.Score,
	basePath string,
	relPath string,
) (err error) {
	return r.createRestorer(r).RestoreDirectory(score, basePath, relPath)
}

// Split out for testability. You should not use this directly.
func NewNonRecursiveDirectoryRestorer(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
	fileRestorer FileRestorer,
	wrapped DirectoryRestorer,
) (restorer DirectoryRestorer, err error) {
	restorer = &dirRestorer{blobStore, fileSystem, fileRestorer, wrapped}
	return
}

type dirRestorer struct {
	blobStore blob.Store
	fileSystem fs.FileSystem
	fileRestorer FileRestorer
	wrapped DirectoryRestorer
}

func (r *dirRestorer) RestoreDirectory(
	score blob.Score,
	basePath string,
	relPath string,
) (err error) {
	err = fmt.Errorf("TODO")
	return
}

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
	"github.com/jacobsa/comeback/sys"
)

////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////

// An object that knows how to restore previously backed up directories.
type DirectoryRestorer interface {
	// Recursively restore a directory based on the listing named by the supplied
	// score. The first call should set basePath to the target directory and
	// relPath to the empty string. The target directory must already exist.
	RestoreDirectory(score blob.Score, basePath, relPath string) (err error)
}

// Create a directory restorer that uses the supplied objects.
func NewDirectoryRestorer(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
	fileRestorer FileRestorer,
) (restorer DirectoryRestorer, err error) {
	userRegistry, err := sys.NewUserRegistry()
	if err != nil {
		err = fmt.Errorf("NewUserRegistry: %v", err)
	}

	groupRegistry, err := sys.NewGroupRegistry()
	if err != nil {
		err = fmt.Errorf("NewGroupRegistry: %v", err)
	}

	createRestorer := func(wrapped DirectoryRestorer) DirectoryRestorer {
		restorer, err := NewNonRecursiveDirectoryRestorer(
			blobStore,
			fileSystem,
			fileRestorer,
			userRegistry,
			groupRegistry,
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
	userRegistry sys.UserRegistry,
	groupRegistry sys.GroupRegistry,
	wrapped DirectoryRestorer,
) (restorer DirectoryRestorer, err error) {
	restorer = &dirRestorer{
		blobStore,
		fileSystem,
		fileRestorer,
		userRegistry,
		groupRegistry,
		wrapped,
	}

	return
}

type dirRestorer struct {
	blobStore    blob.Store
	fileSystem   fs.FileSystem
	fileRestorer FileRestorer
	userRegistry sys.UserRegistry
	groupRegistry sys.GroupRegistry
	wrapped      DirectoryRestorer
}

func (r *dirRestorer) RestoreDirectory(
	score blob.Score,
	basePath string,
	relPath string,
) (err error) {
	err = fmt.Errorf("TODO")
	return
}

func (r *dirRestorer) chooseUserId(
	uid sys.UserId,
	username *string,
) (sys.UserId, error) {
	// If there is no symbolic username, just return the UID.
	if username == nil {
		return uid, nil
	}

	// Attempt to look up the username. If it's not found, return the UID.
	betterUid, err := r.userRegistry.FindByName(*username)

	if _, ok := err.(sys.NotFoundError); ok {
		return uid, nil
	} else if err != nil {
		return 0, fmt.Errorf("userRegistry.FindByName: %v", err)
	}

	return betterUid, nil
}

func (r *dirRestorer) chooseGroupId(
	gid sys.GroupId,
	groupname *string,
) (sys.GroupId, error) {
	// If there is no symbolic groupname, just return the UID.
	if groupname == nil {
		return gid, nil
	}

	// Attempt to look up the groupname. If it's not found, return the UID.
	betterGid, err := r.groupRegistry.FindByName(*groupname)

	if _, ok := err.(sys.NotFoundError); ok {
		return gid, nil
	} else if err != nil {
		return 0, fmt.Errorf("groupRegistry.FindByName: %v", err)
	}

	return betterGid, nil
}

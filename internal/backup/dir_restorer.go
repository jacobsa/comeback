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
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/comeback/internal/sys"
)

////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////

// An object that knows how to restore previously backed up directories.
type DirectoryRestorer interface {
	// Recursively restore a directory based on the listing named by the supplied
	// score. The first call should set basePath to the target directory and
	// relPath to the empty string. The target directory must already exist.
	RestoreDirectory(
		ctx context.Context,
		score blob.Score,
		basePath string,
		relPath string) (err error)
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
	ctx context.Context,
	score blob.Score,
	basePath string,
	relPath string,
) (err error) {
	return r.createRestorer(r).RestoreDirectory(ctx, score, basePath, relPath)
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
	blobStore     blob.Store
	fileSystem    fs.FileSystem
	fileRestorer  FileRestorer
	userRegistry  sys.UserRegistry
	groupRegistry sys.GroupRegistry
	wrapped       DirectoryRestorer
}

func (r *dirRestorer) RestoreDirectory(
	ctx context.Context,
	score blob.Score,
	basePath string,
	relPath string,
) (err error) {
	// Load the appropriate blob.
	blob, err := r.blobStore.Load(ctx, score)
	if err != nil {
		err = fmt.Errorf("Loading blob: %v", err)
		return
	}

	// Parse its contents.
	entries, err := repr.UnmarshalDir(blob)
	if err != nil {
		err = fmt.Errorf("Invalid data in blob: %v", err)
		return
	}

	// Deal with each entry.
	for _, entry := range entries {
		entryRelPath := path.Join(relPath, entry.Name)
		entryFullPath := path.Join(basePath, entryRelPath)

		// Switch on type.
		switch entry.Type {
		case fs.TypeFile:
			// Is this a hard link to another file?
			if entry.HardLinkTarget != nil {
				// Create the hard link.
				targetFullPath := path.Join(basePath, *entry.HardLinkTarget)
				err = r.fileSystem.CreateHardLink(targetFullPath, entryFullPath)
				if err != nil {
					err = fmt.Errorf("CreateHardLink: %v", err)
					return
				}

				// There's nothing else to do for a hard link.
				continue
			}

			// Create the file using its blobs.
			err = r.fileRestorer.RestoreFile(
				ctx,
				entry.Scores,
				entryFullPath,
				entry.Permissions,
			)

			if err != nil {
				err = fmt.Errorf("RestoreFile: %v", err)
				return
			}

		case fs.TypeDirectory:
			// Directory listings should be composed of exactly one score.
			if len(entry.Scores) != 1 {
				err = fmt.Errorf(
					"Expected exactly one score for directory entry: %v",
					entry)

				return
			}

			// Create the directory.
			if err = r.fileSystem.Mkdir(entryFullPath, entry.Permissions); err != nil {
				err = fmt.Errorf("Mkdir: %v", err)
				return
			}

			// Restore to the directory.
			err = r.wrapped.RestoreDirectory(
				ctx,
				entry.Scores[0],
				basePath,
				entryRelPath,
			)

			if err != nil {
				return
			}

		case fs.TypeSymlink:
			err = r.fileSystem.CreateSymlink(
				entry.Target,
				entryFullPath,
				entry.Permissions,
			)

			if err != nil {
				err = fmt.Errorf("CreateSymlink: %v", err)
				return
			}

		case fs.TypeNamedPipe:
			err = r.fileSystem.CreateNamedPipe(entryFullPath, entry.Permissions)
			if err != nil {
				err = fmt.Errorf("CreateNamedPipe: %v", err)
				return
			}

		case fs.TypeBlockDevice:
			err = r.fileSystem.CreateBlockDevice(
				entryFullPath,
				entry.Permissions,
				entry.DeviceNumber,
			)

			if err != nil {
				err = fmt.Errorf("CreateBlockDevice: %v", err)
				return
			}

		case fs.TypeCharDevice:
			err = r.fileSystem.CreateCharDevice(
				entryFullPath,
				entry.Permissions,
				entry.DeviceNumber,
			)

			if err != nil {
				err = fmt.Errorf("CreateCharDevice: %v", err)
				return
			}

		default:
			return fmt.Errorf("Don't know how to deal with entry: %v", entry)
		}

		// Fix ownership.
		var uid sys.UserId
		uid, err = r.chooseUserId(entry.Uid, entry.Username)
		if err != nil {
			err = fmt.Errorf("chooseUserId: %v", err)
			return
		}

		var gid sys.GroupId
		gid, err = r.chooseGroupId(entry.Gid, entry.Groupname)
		if err != nil {
			err = fmt.Errorf("chooseGroupId: %v", err)
			return
		}

		if err = r.fileSystem.Chown(entryFullPath, int(uid), int(gid)); err != nil {
			err = fmt.Errorf("Chown: %v", err)
			return
		}

		// Fix modification time, but not on devices (otherwise we get resource
		// busy errors).
		if entry.Type != fs.TypeBlockDevice && entry.Type != fs.TypeCharDevice {
			err = r.fileSystem.SetModTime(entryFullPath, entry.MTime)
			if err != nil {
				err = fmt.Errorf("SetModTime(%s): %v", entryFullPath, err)
				return
			}
		}
	}

	return nil
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

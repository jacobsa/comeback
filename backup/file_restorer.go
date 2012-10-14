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
	"os"
)

////////////////////////////////////////////////////////////////////////
// Public
////////////////////////////////////////////////////////////////////////

// An object that knows how to restore previously backed up files.
type FileRestorer interface {
	// Restore the contents of the supplied scores to the file at the given path.
	// The file must not already exist.
	RestoreFile(scores []blob.Score, path string, perms os.FileMode) (err error)
}

// Create a file restorer that uses the supplied blob store and file systems.
func NewFileRestorer(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
) (restorer FileRestorer, err error) {
	restorer = &fileRestorer{blobStore, fileSystem}
	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type fileRestorer struct {
	blobStore blob.Store
	fileSystem fs.FileSystem
}

func (r *fileRestorer) RestoreFile(
	scores []blob.Score,
	path string,
	perms os.FileMode,
) (err error) {
	err = fmt.Errorf("TODO")
	return
}

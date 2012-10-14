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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
)

// An object that knows how to restore previously backed up files.
type FileRestorer interface {
	// Restore the contents of the supplied scores to the file at the given path,
	// overwriting its contents if it already exists.
	RestoreFile(scores []blob.Score, path string) (err error)
}

// Create a file restorer that uses the supplied blob store and file systems.
func NewFileRestorer(
	blobStore blob.Store,
	fileSystem fs.FileSystem,
) (restorer FileRestorer, err error)

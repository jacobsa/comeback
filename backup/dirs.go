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

// An object that knows how to save directories to some underlying storage.
type DirectorySaver interface {
	// Recursively save the contents of the supplied directory to the underlying
	// storage, returning the score of a blob representing the directory's
	// listing in a format hat can be recovered with repr.Unmarshal.
	Save(dirpath string) (score blob.Score, err error)
}

type directorySaver struct {
	blobStore blob.Store
	fileSystem fs.FileSystem
	fileSaver FileSaver
	wrapped DirectorySaver
}

func (s *directorySaver) saveDir(parent string, fi os.FileInfo) (fs.DirectoryEntry, error) {
}

func (s *directorySaver) Save(dirpath string) (score blob.Score, err error) {
	// Grab a listing for the directory.
	fileInfos, err := s.FileSystem.ReadDir(dirpath)
	if err != nil {
		return nil, fmt.Errorf("Listing directory: %v", err)
	}

	// Process each entry in the directory, building a list of DirectoryEntry
	// structs.
	entries := []fs.DirectoryEntry{}
	for _, fileInfo := range fileInfos {
		var entry DirectoryEntry

		// Call the appropriate method based on this entry's type.
		switch {
		case fileInfo.IsDir():
			entry, err = s.saveDir(dirpath, fileInfo)
		case fileInfo.Mode() & os.ModeType == 0:
			entry, err = s.saveFile(dirpath, fileInfo)
	  default:
			err = fmt.Errorf("Unhandled mode: %v", fileInfo.Mode())
		}

		if err != nil {
			return nil, err
		}
	}
}

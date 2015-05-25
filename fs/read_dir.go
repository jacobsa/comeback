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

package fs

import (
	"io/ioutil"
	"os"
	"path"
)

func (fs *fileSystem) ReadDir(
	dirpath string) (entries []*DirectoryEntry, err error) {
	// Call ioutil.
	fileInfos, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	// Convert each entry.
	entries = []*DirectoryEntry{}
	for _, fileInfo := range fileInfos {
		entry, err := fs.convertFileInfo(fileInfo)
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

// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package restore

import "github.com/jacobsa/comeback/internal/fs"

// A node within the tree to be restored, rooted at the score of the backup job
// selected by the user.
type node struct {
	// The path of the file (or directory, etc.) relative to the root of the
	// backup.
	RelPath string

	// Type, size, etc. information about the file, directory, etc. Also contains
	// a list of scores from which its contents can be loaded.
	Info fs.FileInfo

	// The nodes comprising the children of this directory. Empty for
	// non-directories.
	Children []*node
}

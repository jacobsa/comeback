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

package save

import (
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/state"
)

// A node within the tree defined by the file system hierarchy rooted at a
// particular directory, called the base path below.
type fsNode struct {
	// The path of the file (or directory, etc.) relative to the base path.
	RelPath string

	// Type, size, etc. information about the file.
	Info fs.FileInfo

	// The node for the parent of this file, or nil if this is the root of the
	// tree of interest.
	Parent *fsNode

	// The nodes comprising the children of this directory. Empty for
	// non-directories.
	Children []*fsNode

	// For private use by consultScoreMap and updateScoreMap: if the scores in
	// Info ought to be inserted into the score map after being computed, the key
	// to use when doing so.
	ScoreMapKey *state.ScoreMapKey
}

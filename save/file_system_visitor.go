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
	"fmt"
	"os"
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/graph"
)

type PathAndFileInfo struct {
	// The absolute path to the file (or directory, etc.).
	Path string

	// Information about the file, as returned by os.Lstat.
	Info os.FileInfo
}

// Create a visitor that walks the directory hierarchy rooted at the given base
// path, excluding any relative path that matches any of the supplied
// exclusions, along with any of its descendents. Everything encountered and
// not excluded will be emitted to the supplied channel in an unspecified
// order. The channel will not be closed.
//
// It is expected that node names are paths relative to the supplied base path.
// In particular, to walk the entire hierarchy, use "" as the traversal root.
func NewFileSystemVisitor(
	basePath string,
	output chan<- PathAndFileInfo) (v graph.Visitor) {
	v = &fileSystemVisitor{
		basePath: basePath,
		output:   output,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type fileSystemVisitor struct {
	basePath string
	output   chan<- PathAndFileInfo
}

func (fsv *fileSystemVisitor) Visit(
	ctx context.Context,
	node string) (adjacent []string, err error) {
	// Open the directory for reading.
	f, err := os.Open(path.Join(fsv.basePath, node))
	if err != nil {
		err = fmt.Errorf("Open: %v", err)
		return
	}

	// Read all of the names in the directory and lstat them.
	entries, err := f.Readdir(0)
	if err != nil {
		err = fmt.Errorf("Readdir: %v", err)
		return
	}

	// Feed to the output channel, returning directories as adjacent nodes that
	// need to be visited.
	for _, fi := range entries {
		// Send to the output channel.
		pfi := PathAndFileInfo{
			Path: path.Join(fsv.basePath, node, fi.Name()),
			Info: fi,
		}

		select {
		// Cancelled?
		case <-ctx.Done():
			err = ctx.Err()
			return

		case fsv.output <- pfi:
		}

		// Record child directories.
		if fi.IsDir() {
			adjacent = append(adjacent, path.Join(node, fi.Name()))
		}
	}

	return
}

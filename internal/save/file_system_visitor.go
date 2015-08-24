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
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
)

type PathAndFileInfo struct {
	// The path to the file (or directory, etc.), relative to the backup base
	// path.
	Path string

	// Information about the file, as returned by os.Lstat.
	Info os.FileInfo
}

// Create a graph.SuccessorFinder that models the directory hierarchy rooted at
// the given base path, excluding any relative path that matches any of the
// supplied exclusions, along with any of its descendents.
//
// The nodes involved are of type *PathAndFileInfo. Special case: a value of
// *PathAndFileInfo(nil) is taken to be the root of the hierarchy.
func NewFileSystemVisitor(
	basePath string,
	exclusions []*regexp.Regexp) (sf graph.SuccessorFinder) {
	sf = &fileSystemVisitor{
		basePath:   basePath,
		exclusions: exclusions,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type fileSystemVisitor struct {
	basePath   string
	exclusions []*regexp.Regexp
}

func (fsv *fileSystemVisitor) FindDirectSuccessors(
	ctx context.Context,
	node graph.Node) (successors []graph.Node, err error) {
	// Ensure the input is of the correct type.
	pfi, ok := node.(*PathAndFileInfo)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", node)
		return
	}

	// Extract the pertinent info.
	var relPath string
	var isDir bool

	if pfi == nil {
		// This is the root of the hierarchy.
		isDir = true
	} else {
		relPath = pfi.Path
		isDir = pfi.Info.IsDir()
	}

	// Skip non-directories; they have no successors.
	if !isDir {
		return
	}

	// Read and lstat all of the names in the directory.
	entries, err := fsv.readDir(relPath)
	if err != nil {
		err = fmt.Errorf("readDir: %v", err)
		return
	}

	// Filter out excluded entries, and return the rest as adjacent nodes.
	for _, fi := range entries {
		childRelPath := path.Join(relPath, fi.Name())
		if fsv.shouldSkip(childRelPath) {
			continue
		}

		successor := &PathAndFileInfo{
			Path: childRelPath,
			Info: fi,
		}

		successors = append(successors, successor)
	}

	return
}

// Read and lstat everything in the directory with the given relative path.
func (fsv *fileSystemVisitor) readDir(
	relPath string) (entries []os.FileInfo, err error) {
	// Open the directory for reading.
	f, err := os.Open(path.Join(fsv.basePath, relPath))
	if err != nil {
		err = fmt.Errorf("Open: %v", err)
		return
	}

	defer func() {
		closeErr := f.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("Close: %v", closeErr)
		}
	}()

	// Read.
	entries, err = f.Readdir(0)
	if err != nil {
		err = fmt.Errorf("Readdir: %v", err)
		return
	}

	return
}

func (fsv *fileSystemVisitor) shouldSkip(relPath string) bool {
	for _, re := range fsv.exclusions {
		if re.MatchString(relPath) {
			return true
		}
	}

	return false
}

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

// Create a graph.SuccessorFinder that models the directory hierarchy rooted at
// the given base path, excluding all relative paths that matches any of the
// supplied exclusions, along with all of their descendants.
//
// The nodes involved are of type *fsNode. The successor finder fills in
// RelPath, Info, and Parent fields.
func newSuccessorFinder(
	basePath string,
	exclusions []*regexp.Regexp) (sf graph.SuccessorFinder) {
	sf = &fsSuccessorFinder{
		basePath:   basePath,
		exclusions: exclusions,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type fsSuccessorFinder struct {
	basePath   string
	exclusions []*regexp.Regexp
}

func (sf *fsSuccessorFinder) FindDirectSuccessors(
	ctx context.Context,
	node graph.Node) (successors []graph.Node, err error) {
	// Ensure the input is of the correct type.
	n, ok := node.(*fsNode)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", node)
		return
	}

	// Skip non-directories; they have no successors.
	if !n.Info.IsDir() {
		return
	}

	// Read and lstat all of the names in the directory.
	entries, err := sf.readDir(n.RelPath)
	if err != nil {
		err = fmt.Errorf("readDir: %v", err)
		return
	}

	// Filter out excluded entries, and return the rest as adjacent nodes.
	for _, fi := range entries {
		childRelPath := path.Join(n.RelPath, fi.Name())
		if sf.shouldSkip(childRelPath) {
			continue
		}

		successor := &fsNode{
			RelPath: childRelPath,
			Info:    fi,
			Parent:  n,
		}

		successors = append(successors, successor)
	}

	return
}

// Read and lstat everything in the directory with the given relative path.
func (sf *fsSuccessorFinder) readDir(
	relPath string) (entries []os.FileInfo, err error) {
	// Open the directory for reading.
	f, err := os.Open(path.Join(sf.basePath, relPath))
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

func (sf *fsSuccessorFinder) shouldSkip(relPath string) bool {
	for _, re := range sf.exclusions {
		if re.MatchString(relPath) {
			return true
		}
	}

	return false
}

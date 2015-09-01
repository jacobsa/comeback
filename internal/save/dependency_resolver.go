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
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
)

// Create a dag.DependencyResolver that models the directory hierarchy rooted
// at the given base path, excluding all relative paths that matches any of the
// supplied exclusions, along with all of their descendants. Children are
// dependencies of parents.
//
// The nodes involved are of type *fsNode. The resolver fills in the RelPath,
// Info and Parent fields of the dependencies, and the Children field of the
// node on which it is called. The Scores field of Info is left as nil,
// however.
//
// Results are guaranteed to be sorted by name, for stability and for
// compatibility with old backup corpora. This is stricter than the
// case-insensitive order apparently automatically offered on OS X.
func newDependencyResolver(
	basePath string,
	exclusions []*regexp.Regexp) (dr dag.DependencyResolver) {
	dr = &dependencyResolver{
		basePath:   basePath,
		exclusions: exclusions,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type dependencyResolver struct {
	basePath   string
	exclusions []*regexp.Regexp
}

func (dr *dependencyResolver) FindDependencies(
	ctx context.Context,
	node dag.Node) (deps []dag.Node, err error) {
	// Ensure the input is of the correct type.
	n, ok := node.(*fsNode)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", node)
		return
	}

	// Skip non-directories; they have no dependencies.
	if n.Info.Type != fs.TypeDirectory {
		return
	}

	// Read and lstat all of the names in the directory.
	listing, err := dr.readDir(n.RelPath)
	if err != nil {
		err = fmt.Errorf("readDir: %v", err)
		return
	}

	// Filter out excluded entries, converting the rest to *fs.FileInfo and
	// returning them as dependencies.
	for _, fi := range listing {
		// Skip?
		childRelPath := path.Join(n.RelPath, fi.Name())
		if dr.shouldSkip(childRelPath) {
			continue
		}

		// Read a symlink target if necesssary.
		var symlinkTarget string
		if fi.Mode()&os.ModeSymlink != 0 {
			symlinkTarget, err = os.Readlink(path.Join(dr.basePath, childRelPath))
			if err != nil {
				err = fmt.Errorf("Readlink: %v", err)
				return
			}
		}

		// Convert.
		var entry *fs.FileInfo
		entry, err = fs.ConvertFileInfo(fi, symlinkTarget)
		if err != nil {
			err = fmt.Errorf("ConvertFileInfo: %v", err)
			return
		}

		// Return a dependency.
		child := &fsNode{
			RelPath: childRelPath,
			Info:    *entry,
			Parent:  n,
		}

		deps = append(deps, child)
		n.Children = append(n.Children, child)
	}

	return
}

// Read and lstat everything in the directory with the given relative path.
// Sort by name.
func (dr *dependencyResolver) readDir(
	relPath string) (entries []os.FileInfo, err error) {
	// Note that ioutil.ReadDir guarantees that the output is sorted by name.
	entries, err = ioutil.ReadDir(path.Join(dr.basePath, relPath))
	return
}

func (dr *dependencyResolver) shouldSkip(relPath string) bool {
	for _, re := range dr.exclusions {
		if re.MatchString(relPath) {
			return true
		}
	}

	return false
}

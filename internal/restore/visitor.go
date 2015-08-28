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

import (
	"fmt"
	"log"
	"os"
	"path"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
)

// Create a dag.Visitor for *node.
//
// For each node n, the visitor does the following:
//
//  *  Ensure that the directory path.Dir(n.RelPath) exists.
//  *  <Perform type-specific action.>
//  *  Set the appropriate permissions and times for n.RelPath.
//
// Restoring ownership is not supported.
//
// The type-specific actions are as follows:
//
//  *  Files: create the file with the contents described by n.Info.Scores.
//  *  Directories: ensure that the directory n.RelPath exists.
//  *  Symlinks: create a symlink pointing at n.Info.Target.
//
func newVisitor(
	basePath string,
	blobStore blob.Store,
	logger *log.Logger) (v dag.Visitor) {
	v = &visitor{
		basePath:  basePath,
		blobStore: blobStore,
		logger:    logger,
	}

	return
}

type visitor struct {
	basePath  string
	blobStore blob.Store
	logger    *log.Logger
}

func (v *visitor) Visit(ctx context.Context, untyped dag.Node) (err error) {
	// Ensure the input is of the correct type.
	n, ok := untyped.(*node)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", untyped)
		return
	}

	absPath := path.Join(v.basePath, n.RelPath)

	// Make sure the leading directories exist so that we can write into them.
	err = os.MkdirAll(path.Dir(absPath), 0700)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %v", err)
		return
	}

	// Perform type-specific logic.
	switch n.Info.Type {
	case fs.TypeSymlink:
		err = v.handleSymlink(absPath, n.Info.Target)
		if err != nil {
			err = fmt.Errorf("handleSymlink: %v", err)
			return
		}

	default:
		err = fmt.Errorf("Unhandled type %d for node: %q", n.Info.Type, n.RelPath)
		return
	}

	// Fix up permissions.
	err = chmod(absPath, n.Info.Permissions)
	if err != nil {
		err = fmt.Errorf("chmod: %v", err)
		return
	}

	// Fix up mtime.
	err = os.Chtimes(absPath, time.Now(), n.Info.MTime)
	if err != nil {
		err = fmt.Errorf("Chtimes: %v", err)
		return
	}

	return
}

func (v *visitor) handleSymlink(absPath string, target string) (err error) {
	err = os.Symlink(target, absPath)
	if err != nil {
		err = fmt.Errorf("os.Symlink: %v", err)
		return
	}

	return
}

// Like os.Chmod, but operates on symlinks rather than their targets.
func chmod(name string, mode os.FileMode) (err error) {
	err = fchmodat(
		AT_FDCWD,
		name,
		uint32(mode.Perm()),
		AT_SYMLINK_NOFOLLOW)

	if err != nil {
		err = fmt.Errorf("fchmodat: %v", err)
		return
	}

	return
}

// Constants missing from package syscall and package unux.
const (
	SYS_FCHMODAT        = 467
	AT_FDCWD            = -2
	AT_SYMLINK_NOFOLLOW = 0x0020
)

// Work around the lack of syscall.Fchmodat.
func fchmodat(
	fd int,
	path string,
	mode uint32,
	flag int) (err error) {
	// Convert to the string format expected by the syscall.
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		err = fmt.Errorf("BytePtrFromString(%q): %v", path, err)
		return
	}

	// Call through.
	_, _, e := syscall.Syscall6(
		SYS_FCHMODAT,
		uintptr(fd),
		uintptr(unsafe.Pointer(p)),
		uintptr(mode),
		uintptr(flag),
		0, 0)

	use(unsafe.Pointer(p))

	if e != 0 {
		err = e
		return
	}

	return
}

// Ensure that the supplied pointer stays alive until the call to use.
// Cf. https://github.com/golang/go/commit/cf622d7
//
//go:noescape
func use(p unsafe.Pointer)

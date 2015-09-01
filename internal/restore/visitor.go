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
	"time"
	"unsafe"

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
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
	case fs.TypeFile:
		err = v.writeFileContents(ctx, absPath, n.Info.Scores)
		if err != nil {
			err = fmt.Errorf("writeFileContents: %v", err)
			return
		}

	case fs.TypeDirectory:
		err = os.MkdirAll(absPath, 0700)
		if err != nil {
			err = fmt.Errorf("MkdirAll (for node): %v", err)
			return
		}

	case fs.TypeSymlink:
		err = os.Symlink(n.Info.Target, absPath)
		if err != nil {
			err = fmt.Errorf("os.Symlink: %v", err)
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
	err = chtimes(absPath, time.Now(), n.Info.MTime)
	if err != nil {
		err = fmt.Errorf("chtimes: %v", err)
		return
	}

	return
}

func (v *visitor) writeFileContents(
	ctx context.Context,
	absPath string,
	scores []blob.Score) (err error) {
	// Create the file.
	f, err := os.OpenFile(
		absPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0400)

	if err != nil {
		err = fmt.Errorf("OpenFile: %v", err)
		return
	}

	defer f.Close()

	// Load and write out each chunk.
	for _, s := range scores {
		var chunk []byte

		// Load.
		chunk, err = v.blobStore.Load(ctx, s)
		if err != nil {
			err = fmt.Errorf("Load(%s): %v", s.Hex(), err)
			return
		}

		// Unmarshal.
		chunk, err = repr.UnmarshalFile(chunk)
		if err != nil {
			err = fmt.Errorf("UnmarshalFile(%s): %v", s.Hex(), err)
			return
		}

		// Write.
		_, err = f.Write(chunk)
		if err != nil {
			err = fmt.Errorf("Write: %v", err)
			return
		}
	}

	// Finish off the file.
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

// Like os.Chmod, but operates on symlinks rather than their targets.
func chmod(name string, mode os.FileMode) (err error) {
	err = fchmodat(
		at_FDCWD,
		name,
		uint32(mode.Perm()),
		at_SYMLINK_NOFOLLOW)

	if err != nil {
		err = fmt.Errorf("fchmodat: %v", err)
		return
	}

	return
}

// Constants missing from package unix.
const (
	at_FDCWD            = -2
	at_SYMLINK_NOFOLLOW = 0x0020
)

// Work around the lack of unix.Fchmodat.
func fchmodat(
	fd int,
	path string,
	mode uint32,
	flag int) (err error) {
	// Convert to the string format expected by the syscall.
	p, err := unix.BytePtrFromString(path)
	if err != nil {
		err = fmt.Errorf("BytePtrFromString(%q): %v", path, err)
		return
	}

	// Call through.
	_, _, e := unix.Syscall6(
		unix.SYS_FCHMODAT,
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

// Like os.Chtimes, but doesn't follow symlinks.
// Cf. http://stackoverflow.com/a/10611073/1505451
func chtimes(path string, atime time.Time, mtime time.Time) (err error) {
	// Open the file without following symlinks.
	fd, err := unix.Open(path, unix.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer unix.Close(fd)

	// Call futimes.
	var utimes [2]unix.Timeval
	utimes[0] = unix.NsecToTimeval(atime.UnixNano())
	utimes[1] = unix.NsecToTimeval(mtime.UnixNano())

	err = unix.Futimes(fd, utimes[:])
	if err != nil {
		err = fmt.Errorf("unix.Futimes: %v", err)
		return
	}

	return nil
}

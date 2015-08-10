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
	"fmt"
	"io"
	"os"
	"syscall"
)

func (fs *fileSystem) CreateNamedPipe(
	path string,
	perms os.FileMode,
) (err error) {
	// Create the pipe.
	if err = syscall.Mkfifo(path, syscallPermissions(perms)); err != nil {
		return
	}

	// Fix any changes to the permissions made by the process's umask value.
	if err = fs.SetPermissions(path, perms); err != nil {
		err = fmt.Errorf("SetPermissions: %v", err)
		return
	}

	return
}

func (fs *fileSystem) CreateBlockDevice(
	path string,
	perms os.FileMode,
	devNum int32,
) (err error) {
	// Create the device.
	mode := syscallPermissions(perms) | syscall.S_IFBLK
	if err = syscall.Mknod(path, mode, int(devNum)); err != nil {
		err = fmt.Errorf("syscall.Mknod: %v", err)
		return
	}

	// Fix any changes to the permissions made by the process's umask value.
	if err = fs.SetPermissions(path, perms); err != nil {
		err = fmt.Errorf("SetPermissions: %v", err)
		return
	}

	return
}

func (fs *fileSystem) CreateCharDevice(
	path string,
	perms os.FileMode,
	devNum int32,
) (err error) {
	// Create the device.
	mode := syscallPermissions(perms) | syscall.S_IFCHR
	if err = syscall.Mknod(path, mode, int(devNum)); err != nil {
		err = fmt.Errorf("syscall.Mknod: %v", err)
		return
	}

	// Fix any changes to the permissions made by the process's umask value.
	if err = fs.SetPermissions(path, perms); err != nil {
		err = fmt.Errorf("SetPermissions: %v", err)
		return
	}

	return nil
}

func (fs *fileSystem) CreateFile(
	path string,
	perms os.FileMode,
) (w io.WriteCloser, err error) {
	// Open the file.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perms)
	if err != nil {
		return
	}

	w = f

	// Fix any changes to the permissions made by the process's umask value.
	if err = fs.setPermissions(int(f.Fd()), perms); err != nil {
		err = fmt.Errorf("setPermissions: %v", err)
		return
	}

	return
}

func (fs *fileSystem) Mkdir(
	path string,
	perms os.FileMode,
) (err error) {
	// Create the directory.
	if err = os.Mkdir(path, perms); err != nil {
		return
	}

	// Fix any changes to the permissions made by the process's umask value.
	if err = fs.SetPermissions(path, perms); err != nil {
		err = fmt.Errorf("SetPermissions: %v", err)
		return
	}

	return
}

func (fs *fileSystem) CreateSymlink(
	target string,
	source string,
	perms os.FileMode,
) (err error) {
	// Create the link.
	if err = os.Symlink(target, source); err != nil {
		return
	}

	// Set the permissions. This is meaningless on POSIX operating systems in
	// general, but OS X lets you do it.
	if err = fs.SetPermissions(source, perms); err != nil {
		err = fmt.Errorf("SetPermissions: %v", err)
		return
	}

	return
}

func (fs *fileSystem) CreateHardLink(
	target string,
	source string,
) (err error) {
	err = os.Link(target, source)
	return
}

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

func (fs *fileSystem) CreateNamedPipe(path string, perms os.FileMode) error {
	return syscall.Mkfifo(path, syscallPermissions(perms))
}

func (fs *fileSystem) CreateBlockDevice(
	path string,
	perms os.FileMode,
	devNum int32) error {
	mode := syscallPermissions(perms) | syscall.S_IFBLK
	if err := syscall.Mknod(path, mode, int(devNum)); err != nil {
		return fmt.Errorf("syscall.Mknod: %v", err)
	}

	return nil
}

func (fs *fileSystem) CreateCharDevice(
	path string,
	perms os.FileMode,
	devNum int32) error {
	mode := syscallPermissions(perms) | syscall.S_IFCHR
	if err := syscall.Mknod(path, mode, int(devNum)); err != nil {
		return fmt.Errorf("syscall.Mknod: %v", err)
	}

	return nil
}

func (fs *fileSystem) CreateFile(
	path string,
	perms os.FileMode,
) (w io.WriteCloser, err error) {
	w, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perms)
	return
}

func (fs *fileSystem) CreateHardLink(target, source string) (err error) {
	err = os.Link(target, source)
	return
}

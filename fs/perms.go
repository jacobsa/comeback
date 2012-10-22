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
	"os"
	"syscall"
)

func syscallPermissions(permissions os.FileMode) (o uint32) {
	// Include r/w/x permission bits.
	o = uint32(permissions & os.ModePerm)

	// Also include setuid/setgid/sticky bits.
	if permissions&os.ModeSetuid != 0 {
		o |= syscall.S_ISUID
	}

	if permissions&os.ModeSetgid != 0 {
		o |= syscall.S_ISGID
	}

	if permissions&os.ModeSticky != 0 {
		o |= syscall.S_ISVTX
	}

	return
}

// Set permissions on the supplied file descriptor.
func (fs *fileSystem) setPermissions(fd int, perms os.FileMode) (err error) {
	mode := syscallPermissions(perms)
	err = syscall.Fchmod(fd, mode)
	return
}

func (fs *fileSystem) SetPermissions(path string, permissions os.FileMode) error {
	// Open the file without following symlinks. Use O_NONBLOCK to allow opening
	// of named pipes without a writer.
	fd, err := syscall.Open(path, syscall.O_NONBLOCK|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Set permissions.
	return fs.setPermissions(fd, permissions)
}

func (fs *fileSystem) Chown(path string, uid int, gid int) (err error) {
	err = os.Lchown(path, uid, gid)
	return
}

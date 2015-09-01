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
	"os"
	"syscall"

	"github.com/jacobsa/comeback/internal/sys"
)

const (
	// A permissions bit mask that matches chmod(2)'s notion of permissions.
	permissionBits os.FileMode = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
)

var gUserRegistry = sys.NewUserRegistry()
var gGroupRegistry = sys.NewGroupRegistry()

// Convert the result of os.Lstat or os.Stat to a local FileInfo struct.
// symlinkTarget should be empty if this is not a symlink.
func ConvertFileInfo(
	in os.FileInfo,
	symlinkTarget string) (out *FileInfo, err error) {
	out, err = convertFileInfo(in, symlinkTarget, gUserRegistry, gGroupRegistry)
	return
}

// Like ConvertFileInfo, but allows injecting registries.
func convertFileInfo(
	in os.FileInfo,
	symlinkTarget string,
	userRegistry sys.UserRegistry,
	groupRegistry sys.GroupRegistry) (out *FileInfo, err error) {
	// Grab system-specific info.
	statT, ok := in.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("Unexpected sys value: %v", in.Sys())
	}

	if statT.Size < 0 {
		panic(fmt.Sprintf("Unexpected size: %d", statT.Size))
	}

	// Create the basic struct.
	out = &FileInfo{
		Name:             in.Name(),
		Permissions:      in.Mode() & permissionBits,
		Uid:              sys.UserId(statT.Uid),
		Gid:              sys.GroupId(statT.Gid),
		MTime:            in.ModTime(),
		Size:             uint64(statT.Size),
		ContainingDevice: statT.Dev,
		Inode:            statT.Ino,
		Target:           symlinkTarget,
	}

	// Attempt to look up user info.
	username, err := userRegistry.FindById(out.Uid)

	if _, ok := err.(sys.NotFoundError); ok {
		err = nil
	} else if err != nil {
		return nil, fmt.Errorf("Looking up user: %v", err)
	} else {
		out.Username = &username
	}

	// Attempt to look up group info.
	groupname, err := groupRegistry.FindById(out.Gid)

	if _, ok := err.(sys.NotFoundError); ok {
		err = nil
	} else if err != nil {
		return nil, fmt.Errorf("Looking up group: %v", err)
	} else {
		out.Groupname = &groupname
	}

	// Convert the type.
	typeBits := in.Mode() & (os.ModeType | os.ModeCharDevice)
	switch typeBits {
	case 0:
		out.Type = TypeFile
	case os.ModeDir:
		out.Type = TypeDirectory
	case os.ModeSymlink:
		out.Type = TypeSymlink
	case os.ModeDevice:
		out.Type = TypeBlockDevice
		out.DeviceNumber = statT.Rdev
	case os.ModeDevice | os.ModeCharDevice:
		out.Type = TypeCharDevice
		out.DeviceNumber = statT.Rdev
	case os.ModeNamedPipe:
		out.Type = TypeNamedPipe
	case os.ModeSocket:
		out.Type = TypeSocket
	default:
		return out, fmt.Errorf("Unhandled mode: %v %u", in.Mode(), in.Mode())
	}

	return out, nil
}

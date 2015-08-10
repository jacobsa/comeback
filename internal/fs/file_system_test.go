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

package fs_test

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"syscall"
	"testing"

	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/sys"
	"github.com/jacobsa/comeback/internal/sys/group"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestFileSystemTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type fileSystemTest struct {
	mockController          oglemock.Controller
	userRegistry            sys.UserRegistry
	groupRegistry           sys.GroupRegistry
	fileSystem              fs.FileSystem
	baseDir                 string
	baseDirContainingDevice int32
	baseDirInode            uint64
	myUid                   sys.UserId
	myUsername              string
	myGid                   sys.GroupId
	myGroupname             string
}

func (t *fileSystemTest) setUpFileSystem() {
	var err error
	if t.fileSystem, err = fs.NewFileSystem(t.userRegistry, t.groupRegistry); err != nil {
		log.Fatalf("Creating file system: %v", err)
	}
}

func (t *fileSystemTest) SetUp(i *TestInfo) {
	var err error

	// Set up dependencies.
	t.mockController = i.MockController

	if t.userRegistry, err = sys.NewUserRegistry(); err != nil {
		log.Fatalf("Creating user registry: %v", err)
	}

	if t.groupRegistry, err = sys.NewGroupRegistry(); err != nil {
		log.Fatalf("Creating group registry: %v", err)
	}

	// Set up the file system.
	t.setUpFileSystem()

	// Create a temporary directory.
	t.baseDir, err = ioutil.TempDir("", "ReadDirTest_")
	if err != nil {
		log.Fatalf("Creating baseDir: %v", err)
	}

	// Grab device and inode info for the directory.
	fi, err := os.Stat(t.baseDir)
	if err != nil {
		log.Fatalf("Statting baseDir: %v", err)
	}

	sysInfo, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		log.Fatalf("Bad sys info: %v", fi.Sys())
	}

	t.baseDirContainingDevice = sysInfo.Dev
	t.baseDirInode = sysInfo.Ino

	AssertNe(0, t.baseDirContainingDevice)
	AssertNe(0, t.baseDirInode)

	// Find user info.
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("Getting current user: %v", err)
	}

	currentUserId, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		log.Fatalf("Invalid UID: %s", currentUser.Uid)
	}

	t.myUid = sys.UserId(currentUserId)
	t.myUsername = currentUser.Username

	AssertNe(0, t.myUid)
	AssertNe("", t.myUsername)

	// Find group info.
	currentGroup, err := group.Current()
	if err != nil {
		log.Fatalf("Getting current group: %v", err)
	}

	currentGroupId, err := strconv.Atoi(currentGroup.Gid)
	if err != nil {
		log.Fatalf("Invalid GID: %s", currentGroup.Gid)
	}

	t.myGid = sys.GroupId(currentGroupId)
	t.myGroupname = currentGroup.Groupname

	AssertNe(0, t.myGid)
	AssertNe("", t.myGroupname)
}

func (t *fileSystemTest) TearDown() {
	err := os.RemoveAll(t.baseDir)
	if err != nil {
		log.Fatalf("Couldn't remove: %s", t.baseDir)
	}
}

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
	"errors"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/sys"
	"github.com/jacobsa/comeback/sys/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"
	"time"
)

func TestStat(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type StatTest struct {
	fileSystemTest

	path string
	entry fs.DirectoryEntry
	err error
}

func init() { RegisterTestSuite(&StatTest{}) }

func (t *StatTest) call() {
	t.entry, t.err = t.fileSystem.Stat(t.path)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StatTest) NonExistentPath() {
	t.path = path.Join(t.baseDir, "foobar")

	t.call()

	ExpectThat(t.err, Error(HasSubstr("lstat")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *StatTest) ErrorLookingUpOwnerId() {
	var err error

	// Create a mock user registry, and a new file system that uses it.
	mockRegistry := mock_sys.NewMockUserRegistry(t.mockController, "registry")
	t.userRegistry = mockRegistry
	t.setUpFileSystem()

	// Create a file.
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte(""), 0600)
	AssertEq(nil, err)

	// Registry
	ExpectCall(mockRegistry, "FindById")(t.myUid).
		WillOnce(oglemock.Return("", errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Looking up")))
	ExpectThat(t.err, Error(HasSubstr("user")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *StatTest) ErrorLookingUpGroupId() {
	var err error

	// Create a mock group registry, and a new file system that uses it.
	mockRegistry := mock_sys.NewMockGroupRegistry(t.mockController, "registry")
	t.groupRegistry = mockRegistry
	t.setUpFileSystem()

	// Create a file.
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte(""), 0600)
	AssertEq(nil, err)

	// Registry
	ExpectCall(mockRegistry, "FindById")(t.myGid).
		WillOnce(oglemock.Return("", errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Looking up")))
	ExpectThat(t.err, Error(HasSubstr("group")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *StatTest) UnknownOwnerId() {
	var err error

	// Create a mock user registry, and a new file system that uses it.
	mockRegistry := mock_sys.NewMockUserRegistry(t.mockController, "registry")
	t.userRegistry = mockRegistry
	t.setUpFileSystem()

	// Create a file.
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte(""), 0600)
	AssertEq(nil, err)

	// Registry
	ExpectCall(mockRegistry, "FindById")(t.myUid).
		WillOnce(oglemock.Return("", sys.NotFoundError("taco")))

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(t.myUid, t.entry.Uid)
	ExpectEq(nil, t.entry.Username)
}

func (t *StatTest) UnknownGroupId() {
	var err error

	// Create a mock group registry, and a new file system that uses it.
	mockRegistry := mock_sys.NewMockGroupRegistry(t.mockController, "registry")
	t.groupRegistry = mockRegistry
	t.setUpFileSystem()

	// Create a file.
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte(""), 0600)
	AssertEq(nil, err)

	// Registry
	ExpectCall(mockRegistry, "FindById")(t.myGid).
		WillOnce(oglemock.Return("", sys.NotFoundError("taco")))

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(t.myGid, t.entry.Gid)
	ExpectEq(nil, t.entry.Groupname)
}

func (t *StatTest) RegularFile() {
	var err error

	// File
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte("queso"), 0600)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeFile, t.entry.Type)
	ExpectEq("burrito.txt", t.entry.Name)
	ExpectEq("", t.entry.Target)
	ExpectEq(0, t.entry.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, t.entry.Permissions)
	ExpectEq(t.myUid, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectEq(len("queso"), t.entry.Size)
	ExpectThat(t.entry.Scores, ElementsAre())

	AssertNe(0, t.entry.Inode)
	ExpectEq(t.baseDirContainingDevice, t.entry.ContainingDevice)
	ExpectNe(t.baseDirInode, t.entry.Inode)
}

func (t *StatTest) Directory() {
	var err error

	// Dir
	t.path = path.Join(t.baseDir, "burrito")
	err = os.Mkdir(t.path, 0700)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0751|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeDirectory, t.entry.Type)
	ExpectEq("burrito", t.entry.Name)
	ExpectEq("", t.entry.Target)
	ExpectEq(0, t.entry.DeviceNumber)
	ExpectEq(0751|os.ModeSetgid, t.entry.Permissions)
	ExpectEq(t.myUid, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectThat(t.entry.Scores, ElementsAre())
}

func (t *StatTest) Symlinks() {
	var err error

	// Link
	t.path = path.Join(t.baseDir, "burrito")
	err = os.Symlink("/foo/burrito", t.path)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeSymlink, t.entry.Type)
	ExpectEq("burrito", t.entry.Name)
	ExpectEq("/foo/burrito", t.entry.Target)
	ExpectEq(0, t.entry.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, t.entry.Permissions)
	ExpectEq(t.myUid, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectThat(t.entry.Scores, ElementsAre())
}

func (t *StatTest) CharDevices() {
	t.path = "/dev/urandom"

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeCharDevice, t.entry.Type)
	ExpectEq("urandom", t.entry.Name)
	ExpectEq("", t.entry.Target)
	ExpectEq(os.FileMode(0666), t.entry.Permissions)
	ExpectEq(0, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals("root")))
	ExpectEq(0, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals("wheel")))
	ExpectGe(time.Since(t.entry.MTime), 0)
	ExpectLt(time.Since(t.entry.MTime), 365*24*time.Hour)
}

func (t *StatTest) BlockDevices() {
	t.path = "/dev/disk0"

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeBlockDevice, t.entry.Type)
	ExpectEq("disk0", t.entry.Name)
	ExpectEq("", t.entry.Target)
	ExpectEq(os.FileMode(0640), t.entry.Permissions)
	ExpectEq(0, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals("root")))
	ExpectEq(5, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals("operator")))
	ExpectGe(time.Since(t.entry.MTime), 0)
	ExpectLt(time.Since(t.entry.MTime), 365*24*time.Hour)
}

func (t *StatTest) NamedPipes() {
	var err error

	// Pipe
	t.path = path.Join(t.baseDir, "burrito")
	err = makeNamedPipe(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)

	ExpectEq(fs.TypeNamedPipe, t.entry.Type)
	ExpectEq("burrito", t.entry.Name)
	ExpectEq("", t.entry.Target)
	ExpectEq(0714|os.ModeSetgid, t.entry.Permissions)
	ExpectEq(t.myUid, t.entry.Uid)
	ExpectThat(t.entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(t.entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectThat(t.entry.Scores, ElementsAre())
}

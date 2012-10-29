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
	dirpath := path.Join(t.baseDir, "foobar")

	t.call()

	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *StatTest) NotADirectory() {
	dirpath := path.Join(t.baseDir, "foo.txt")
	err := ioutil.WriteFile(dirpath, []byte("foo"), 0400)
	AssertEq(nil, err)

	t.call()

	ExpectThat(t.err, Error(HasSubstr("readdirent")))
	ExpectThat(t.err, Error(HasSubstr("invalid argument")))
}

func (t *StatTest) NoReadPermissions() {
	dirpath := path.Join(t.baseDir, "foo")
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	t.call()

	ExpectThat(t.err, Error(HasSubstr("permission")))
	ExpectThat(t.err, Error(HasSubstr("denied")))
}

func (t *StatTest) ErrorLookingUpOwnerId() {
	var err error

	// Create a mock user registry, and a new file system that uses it.
	mockRegistry := mock_sys.NewMockUserRegistry(t.mockController, "registry")
	t.userRegistry = mockRegistry
	t.setUpFileSystem()

	// Create a file.
	path0 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
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
	path0 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
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
	path0 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
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
	path0 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
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
	var entry *fs.DirectoryEntry

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
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectEq(len("queso"), t.entry.Size)
	ExpectThat(entry.Scores, ElementsAre())

	AssertNe(0, t.entry.Inode)
	ExpectEq(t.baseDirContainingDevice, t.entry.ContainingDevice)
	ExpectNe(t.baseDirInode, t.entry.Inode)
}

func (t *StatTest) Directory() {
	var err error
	var entry *fs.DirectoryEntry

	// Dir
	path := path.Join(t.baseDir, "burrito")
	err = os.Mkdir(path, 0700)
	AssertEq(nil, err)

	err = setPermissions(path, 0751|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(path, mtime)
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
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime), "%v", t.entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())
}

func (t *StatTest) Symlinks() {
	var err error
	var entry *fs.DirectoryEntry

	// Link 0
	path0 := path.Join(t.baseDir, "burrito")
	err = os.Symlink("/foo/burrito", path0)
	AssertEq(nil, err)

	err = setPermissions(path0, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime0 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(path0, mtime0)
	AssertEq(nil, err)

	// Link 1
	path1 := path.Join(t.baseDir, "enchilada")
	err = os.Symlink("/foo/enchilada", path1)
	AssertEq(nil, err)

	err = setPermissions(path1, 0454|syscall.S_ISVTX|syscall.S_ISUID)
	AssertEq(nil, err)

	mtime1 := time.Date(1985, time.March, 18, 15, 33, 0, 0, time.Local)
	err = setModTime(path1, mtime1)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)
	AssertThat(entries, ElementsAre(Any(), Any()))

	entry = entries[0]
	ExpectEq(fs.TypeSymlink, entry.Type)
	ExpectEq("burrito", entry.Name)
	ExpectEq("/foo/burrito", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime0), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	entry = entries[1]
	ExpectEq(fs.TypeSymlink, entry.Type)
	ExpectEq("enchilada", entry.Name)
	ExpectEq("/foo/enchilada", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0454|os.ModeSetuid|os.ModeSticky, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime1), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())
}

func (t *StatTest) CharDevices() {
	var err error

	// Call
	t.call()

	AssertEq(nil, err)

	entry := findEntry(entries, "urandom")
	AssertNe(nil, entry)
	ExpectEq(fs.TypeCharDevice, entry.Type)
	ExpectEq("urandom", entry.Name)
	ExpectEq("", entry.Target)
	urandomDevNumber := entry.DeviceNumber
	ExpectEq(os.FileMode(0666), entry.Permissions)
	ExpectEq(0, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals("root")))
	ExpectEq(0, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals("wheel")))
	ExpectGe(time.Since(entry.MTime), 0)
	ExpectLt(time.Since(entry.MTime), 365*24*time.Hour)

	entry = findEntry(entries, "random")
	AssertNe(nil, entry)
	randomDevNumber := entry.DeviceNumber

	ExpectNe(urandomDevNumber, randomDevNumber)
}

func (t *StatTest) BlockDevices() {
	var err error

	// Call
	t.call()

	AssertEq(nil, err)

	entry := findEntry(entries, "disk0")
	AssertNe(nil, entry)
	ExpectEq(fs.TypeBlockDevice, entry.Type)
	ExpectEq("disk0", entry.Name)
	ExpectEq("", entry.Target)
	disk0DevNumber := entry.DeviceNumber
	ExpectEq(os.FileMode(0640), entry.Permissions)
	ExpectEq(0, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals("root")))
	ExpectEq(5, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals("operator")))
	ExpectGe(time.Since(entry.MTime), 0)
	ExpectLt(time.Since(entry.MTime), 365*24*time.Hour)

	entry = findEntry(entries, "disk0s1")
	AssertNe(nil, entry)
	disk0s1DevNumber := entry.DeviceNumber

	ExpectNe(disk0DevNumber, disk0s1DevNumber)
}

func (t *StatTest) NamedPipes() {
	var err error
	var entry *fs.DirectoryEntry

	// Pipe 0
	path0 := path.Join(t.baseDir, "burrito")
	err = makeNamedPipe(path0, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime0 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(path0, mtime0)
	AssertEq(nil, err)

	// Pipe 1
	path1 := path.Join(t.baseDir, "enchilada")
	err = makeNamedPipe(path1, 0454|syscall.S_ISVTX|syscall.S_ISUID)
	AssertEq(nil, err)

	mtime1 := time.Date(1985, time.March, 18, 15, 33, 0, 0, time.Local)
	err = setModTime(path1, mtime1)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)
	AssertThat(entries, ElementsAre(Any(), Any()))

	entry = entries[0]
	ExpectEq(fs.TypeNamedPipe, entry.Type)
	ExpectEq("burrito", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0714|os.ModeSetgid, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime0), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	entry = entries[1]
	ExpectEq(fs.TypeNamedPipe, entry.Type)
	ExpectEq("enchilada", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0454|os.ModeSetuid|os.ModeSticky, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime1), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())
}

func (t *StatTest) SortsByName() {
	var err error

	// File 0
	path0 := path.Join(t.baseDir, "enchilada")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
	AssertEq(nil, err)

	// File 1
	path1 := path.Join(t.baseDir, "burrito")
	err = ioutil.WriteFile(path1, []byte(""), 0600)
	AssertEq(nil, err)

	// File 2
	path2 := path.Join(t.baseDir, "taco")
	err = ioutil.WriteFile(path2, []byte(""), 0600)
	AssertEq(nil, err)

	// Call
	t.call()

	AssertEq(nil, t.err)
	AssertThat(entries, ElementsAre(Any(), Any(), Any()))

	ExpectEq("burrito", entries[0].Name)
	ExpectEq("enchilada", entries[1].Name)
	ExpectEq("taco", entries[2].Name)
}

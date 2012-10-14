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

func TestReadDir(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func findEntry(entries []*fs.DirectoryEntry, name string) *fs.DirectoryEntry {
	for _, entry := range entries {
		if entry.Name == name {
			return entry
		}
	}

	return nil
}

// Set the modification time for the supplied path without following symlinks
// (as syscall.Chtimes and therefore os.Chtimes do).
//
// c.f. http://stackoverflow.com/questions/10608724/set-modification-date-on-symbolic-link-in-cocoa
func setModTime(path string, mtime time.Time) error {
	// Open the file without following symlinks. Use O_NONBLOCK to allow opening
	// of named pipes without a writer.
	fd, err := syscall.Open(path, syscall.O_NONBLOCK|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call futimes.
	var utimes [2]syscall.Timeval
	atime := time.Now()
	atime_ns := atime.UnixNano()
	mtime_ns := mtime.UnixNano()
	utimes[0] = syscall.NsecToTimeval(atime_ns)
	utimes[1] = syscall.NsecToTimeval(mtime_ns)

	err = syscall.Futimes(fd, utimes[0:])
	if err != nil {
		return err
	}

	return nil
}

// Like os.Chmod, but don't follow symlinks.
func setPermissions(path string, permissions uint32) error {
	// Open the file without following symlinks.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call fchmod.
	err = syscall.Fchmod(fd, permissions)
	if err != nil {
		return err
	}

	return nil
}

// Create a named pipe at the supplied path.
func makeNamedPipe(path string, permissions uint32) error {
	return syscall.Mkfifo(path, permissions)
}

////////////////////////////////////////////////////////////////////////
// ReadDir
////////////////////////////////////////////////////////////////////////

type ReadDirTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&ReadDirTest{}) }

func (t *ReadDirTest) NonExistentPath() {
	dirpath := path.Join(t.baseDir, "foobar")

	_, err := t.fileSystem.ReadDir(dirpath)
	ExpectThat(err, Error(HasSubstr("no such")))
}

func (t *ReadDirTest) NotADirectory() {
	dirpath := path.Join(t.baseDir, "foo.txt")
	err := ioutil.WriteFile(dirpath, []byte("foo"), 0400)
	AssertEq(nil, err)

	_, err = t.fileSystem.ReadDir(dirpath)
	ExpectThat(err, Error(HasSubstr("readdirent")))
	ExpectThat(err, Error(HasSubstr("invalid argument")))
}

func (t *ReadDirTest) NoReadPermissions() {
	dirpath := path.Join(t.baseDir, "foo")
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	_, err = t.fileSystem.ReadDir(dirpath)
	ExpectThat(err, Error(HasSubstr("permission")))
	ExpectThat(err, Error(HasSubstr("denied")))
}

func (t *ReadDirTest) ErrorLookingUpOwnerId() {
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
	_, err = t.fileSystem.ReadDir(t.baseDir)
	ExpectThat(err, Error(HasSubstr("Looking up")))
	ExpectThat(err, Error(HasSubstr("user")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadDirTest) ErrorLookingUpGroupId() {
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
	_, err = t.fileSystem.ReadDir(t.baseDir)
	ExpectThat(err, Error(HasSubstr("Looking up")))
	ExpectThat(err, Error(HasSubstr("group")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ReadDirTest) UnknownOwnerId() {
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
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any()))

	ExpectEq(t.myUid, entries[0].Uid)
	ExpectEq(nil, entries[0].Username)
}

func (t *ReadDirTest) UnknownGroupId() {
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
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any()))

	ExpectEq(t.myGid, entries[0].Gid)
	ExpectEq(nil, entries[0].Groupname)
}

func (t *ReadDirTest) RegularFiles() {
	var err error
	var entry *fs.DirectoryEntry

	// File 0
	path0 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path0, []byte(""), 0600)
	AssertEq(nil, err)

	err = setPermissions(path0, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime0 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(path0, mtime0)
	AssertEq(nil, err)

	// File 1
	path1 := path.Join(t.baseDir, "enchilada.txt")
	err = ioutil.WriteFile(path1, []byte(""), 0600)
	AssertEq(nil, err)

	err = setPermissions(path1, 0454|syscall.S_ISVTX|syscall.S_ISUID)
	AssertEq(nil, err)

	mtime1 := time.Date(1985, time.March, 18, 15, 33, 0, 0, time.Local)
	err = setModTime(path1, mtime1)
	AssertEq(nil, err)

	// Call
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any(), Any()))

	// Entry 0
	entry = entries[0]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("burrito.txt", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime0), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	AssertNe(0, entry.Inode)
	ExpectEq(t.baseDirContainingDevice, entry.ContainingDevice)
	ExpectNe(t.baseDirInode, entry.Inode)

	// Entry 1
	entry = entries[1]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("enchilada.txt", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0454|os.ModeSetuid|os.ModeSticky, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime1), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	AssertNe(0, entry.Inode)
	ExpectEq(t.baseDirContainingDevice, entry.ContainingDevice)
	ExpectNe(t.baseDirInode, entry.Inode)
}

func (t *ReadDirTest) Directories() {
	var err error
	var entry *fs.DirectoryEntry

	// Dir 0
	path0 := path.Join(t.baseDir, "burrito")
	err = os.Mkdir(path0, 0700)
	AssertEq(nil, err)

	err = setPermissions(path0, 0751|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime0 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(path0, mtime0)
	AssertEq(nil, err)

	// Dir 1
	path1 := path.Join(t.baseDir, "enchilada")
	err = os.Mkdir(path1, 0700)
	AssertEq(nil, err)

	err = setPermissions(path1, 0711|syscall.S_ISVTX|syscall.S_ISUID)
	AssertEq(nil, err)

	mtime1 := time.Date(1985, time.March, 18, 15, 33, 0, 0, time.Local)
	err = setModTime(path1, mtime1)
	AssertEq(nil, err)

	// Call
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any(), Any()))

	entry = entries[0]
	ExpectEq(fs.TypeDirectory, entry.Type)
	ExpectEq("burrito", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0751|os.ModeSetgid, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime0), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	entry = entries[1]
	ExpectEq(fs.TypeDirectory, entry.Type)
	ExpectEq("enchilada", entry.Name)
	ExpectEq("", entry.Target)
	ExpectEq(0, entry.DeviceNumber)
	ExpectEq(0711|os.ModeSticky|os.ModeSetuid, entry.Permissions)
	ExpectEq(t.myUid, entry.Uid)
	ExpectThat(entry.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, entry.Gid)
	ExpectThat(entry.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(entry.MTime.Equal(mtime1), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())
}

func (t *ReadDirTest) Symlinks() {
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
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
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

func (t *ReadDirTest) CharDevices() {
	var err error

	// Call
	entries, err := t.fileSystem.ReadDir("/dev")
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

func (t *ReadDirTest) BlockDevices() {
	var err error

	// Call
	entries, err := t.fileSystem.ReadDir("/dev")
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

func (t *ReadDirTest) NamedPipes() {
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
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
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

func (t *ReadDirTest) SortsByName() {
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
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any(), Any(), Any()))

	ExpectEq("burrito", entries[0].Name)
	ExpectEq("enchilada", entries[1].Name)
	ExpectEq("taco", entries[2].Name)
}
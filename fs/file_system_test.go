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
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/sys"
	"github.com/jacobsa/comeback/sys/group"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"
	"testing"
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

////////////////////////////////////////////////////////////////////////
// OpenForReading
////////////////////////////////////////////////////////////////////////

type OpenForReadingTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&OpenForReadingTest{}) }

func (t *OpenForReadingTest) NonExistentFile() {
	filepath := path.Join(t.baseDir, "foobar")

	_, err := t.fileSystem.OpenForReading(filepath)
	ExpectThat(err, Error(HasSubstr("no such")))
}

func (t *OpenForReadingTest) NoReadPermissions() {
	filepath := path.Join(t.baseDir, "foo.txt")
	err := ioutil.WriteFile(filepath, []byte("foo"), 0300)
	AssertEq(nil, err)

	_, err = t.fileSystem.OpenForReading(filepath)
	ExpectThat(err, Error(HasSubstr("permission")))
	ExpectThat(err, Error(HasSubstr("denied")))
}

func (t *OpenForReadingTest) EmptyFile() {
	filepath := path.Join(t.baseDir, "foo.txt")
	contents := []byte{}
	err := ioutil.WriteFile(filepath, contents, 0400)
	AssertEq(nil, err)

	f, err := t.fileSystem.OpenForReading(filepath)
	AssertEq(nil, err)
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectThat(data, DeepEquals(contents))
}

func (t *OpenForReadingTest) FileWithContents() {
	filepath := path.Join(t.baseDir, "foo.txt")
	contents := []byte{0xde, 0xad, 0xbe, 0xef}
	err := ioutil.WriteFile(filepath, contents, 0400)
	AssertEq(nil, err)

	f, err := t.fileSystem.OpenForReading(filepath)
	AssertEq(nil, err)
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	AssertEq(nil, err)
	ExpectThat(data, DeepEquals(contents))
}

////////////////////////////////////////////////////////////////////////
// WriteFile
////////////////////////////////////////////////////////////////////////

type WriteFileTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&WriteFileTest{}) }

func (t *WriteFileTest) ParentDoesntExist() {
	filepath := path.Join(t.baseDir, "foo", "bar")
	data := []byte{}

	err := t.fileSystem.WriteFile(filepath, data, 0600)
	ExpectThat(err, Error(HasSubstr("foo/bar")))
	ExpectThat(err, Error(HasSubstr("no such")))
}

func (t *WriteFileTest) NoWritePermissionsForParent() {
	parent := path.Join(t.baseDir, "foo")
	err := os.Mkdir(parent, 0555)
	AssertEq(nil, err)

	filepath := path.Join(parent, "bar")
	data := []byte{}

	err = t.fileSystem.WriteFile(filepath, data, 0600)
	ExpectThat(err, Error(HasSubstr("permission")))
}

func (t *WriteFileTest) NoWritePermissionsForFile() {
	filepath := path.Join(t.baseDir, "foo.txt")
	err := ioutil.WriteFile(filepath, []byte(""), 0400)
	AssertEq(nil, err)

	data := []byte{}

	err = t.fileSystem.WriteFile(filepath, data, 0600)
	ExpectThat(err, Error(HasSubstr("permission")))
}

func (t *WriteFileTest) AlreadyExists() {
	// Create the file.
	filepath := path.Join(t.baseDir, "foo.txt")
	err := ioutil.WriteFile(filepath, []byte("blahblah"), 0600)
	AssertEq(nil, err)

	// Write it again.
	data := []byte("taco")
	err = t.fileSystem.WriteFile(filepath, data, 0644)
	AssertEq(nil, err)

	// Check its contents.
	contents, err := ioutil.ReadFile(filepath)
	AssertEq(nil, err)
	ExpectThat(contents, DeepEquals(data))
}

func (t *WriteFileTest) DoesntYetExist() {
	// Write the file.
	filepath := path.Join(t.baseDir, "foo.txt")
	data := []byte("taco")
	err := t.fileSystem.WriteFile(filepath, data, 0641)
	AssertEq(nil, err)

	// Check its contents.
	contents, err := ioutil.ReadFile(filepath)
	AssertEq(nil, err)
	ExpectThat(contents, DeepEquals(data))

	// Check its permissions.
	fi, err := os.Stat(filepath)
	AssertEq(nil, err)
	ExpectEq(0641, fi.Mode().Perm())
}

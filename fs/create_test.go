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
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestCreate(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// CreateFile
////////////////////////////////////////////////////////////////////////

type CreateFileTest struct {
	fileSystemTest

	path  string
	perms os.FileMode

	w   io.WriteCloser
	err error
}

func init() { RegisterTestSuite(&CreateFileTest{}) }

func (t *CreateFileTest) SetUp(i *TestInfo) {
	// Common
	t.fileSystemTest.SetUp(i)

	// Set up defaults.
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0644
}

func (t *CreateFileTest) call() {
	t.w, t.err = t.fileSystem.CreateFile(t.path, t.perms)
}

func (t *CreateFileTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

func (t *CreateFileTest) NonExistentParent() {
	t.path = "/foo/bar/baz/qux"

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("qux")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *CreateFileTest) NoPermissionsForParent() {
	dirpath := path.Join(t.baseDir, "foo")
	t.path = path.Join(dirpath, "taco")

	// Parent
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("foo")))
	ExpectThat(t.err, Error(HasSubstr("permission denied")))
}

func (t *CreateFileTest) FileAlreadyExists() {
	// Create
	err := ioutil.WriteFile(t.path, []byte{}, 0644)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("file exists")))
}

func (t *CreateFileTest) CreatesCorrectEntry() {
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0674  // Conflicts with default umask

	// Call
	t.call()
	AssertEq(nil, t.err)
	defer t.w.Close()

	// List
	entries := t.list()

	AssertThat(entries, ElementsAre(Any()))
	entry := entries[0]

	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("taco", entry.Name)
	ExpectEq(0674, entry.Permissions)
}

func (t *CreateFileTest) SavesDataToCorrectPlace() {
	// Call
	t.call()
	AssertEq(nil, t.err)

	// Write
	expected := []byte("taco")
	_, err := t.w.Write(expected)
	AssertEq(nil, err)

	// Close
	AssertEq(nil, t.w.Close())

	// Read
	data, err := ioutil.ReadFile(t.path)
	AssertEq(nil, err)

	ExpectThat(data, DeepEquals(expected))
}

////////////////////////////////////////////////////////////////////////
// Mkdir
////////////////////////////////////////////////////////////////////////

type MkdirTest struct {
	fileSystemTest

	path  string
	perms os.FileMode

	err error
}

func init() { RegisterTestSuite(&MkdirTest{}) }

func (t *MkdirTest) SetUp(i *TestInfo) {
	// Common
	t.fileSystemTest.SetUp(i)

	// Set up defaults.
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0644
}

func (t *MkdirTest) call() {
	t.err = t.fileSystem.Mkdir(t.path, t.perms)
}

func (t *MkdirTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

func (t *MkdirTest) NonExistentParent() {
	t.path = "/foo/bar/baz/qux"

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("qux")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *MkdirTest) NoPermissionsForParent() {
	dirpath := path.Join(t.baseDir, "foo")
	t.path = path.Join(dirpath, "taco")

	// Parent
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("foo")))
	ExpectThat(t.err, Error(HasSubstr("permission denied")))
}

func (t *MkdirTest) FileAlreadyExistsWithSameName() {
	// Create
	err := ioutil.WriteFile(t.path, []byte{}, 0644)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("file exists")))
}

func (t *MkdirTest) CreatesCorrectEntry() {
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0674  // Conflicts with default umask

	// Call
	t.call()
	AssertEq(nil, t.err)

	// List
	entries := t.list()

	AssertThat(entries, ElementsAre(Any()))
	entry := entries[0]

	ExpectEq(fs.TypeDirectory, entry.Type)
	ExpectEq("taco", entry.Name)
	ExpectEq(0674, entry.Permissions)
}

////////////////////////////////////////////////////////////////////////
// CreateNamedPipe
////////////////////////////////////////////////////////////////////////

type CreateNamedPipeTest struct {
	fileSystemTest

	path  string
	perms os.FileMode

	err error
}

func init() { RegisterTestSuite(&CreateNamedPipeTest{}) }

func (t *CreateNamedPipeTest) SetUp(i *TestInfo) {
	// Common
	t.fileSystemTest.SetUp(i)

	// Set up defaults.
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0644
}

func (t *CreateNamedPipeTest) call() {
	t.err = t.fileSystem.CreateNamedPipe(t.path, t.perms)
}

func (t *CreateNamedPipeTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

func (t *CreateNamedPipeTest) NonExistentParent() {
	t.path = "/foo/bar/baz/qux"

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("qux")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *CreateNamedPipeTest) NoPermissionsForParent() {
	dirpath := path.Join(t.baseDir, "foo")
	t.path = path.Join(dirpath, "taco")

	// Parent
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("foo")))
	ExpectThat(t.err, Error(HasSubstr("permission denied")))
}

func (t *CreateNamedPipeTest) FileAlreadyExistsWithSameName() {
	// Create
	err := ioutil.WriteFile(t.path, []byte{}, 0644)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("file exists")))
}

func (t *CreateNamedPipeTest) CreatesCorrectEntry() {
	t.path = path.Join(t.baseDir, "taco")
	t.perms = 0674  // Conflicts with default umask

	// Call
	t.call()
	AssertEq(nil, t.err)

	// List
	entries := t.list()

	AssertThat(entries, ElementsAre(Any()))
	entry := entries[0]

	ExpectEq(fs.TypeNamedPipe, entry.Type)
	ExpectEq("taco", entry.Name)
	ExpectEq(0674, entry.Permissions)
}

////////////////////////////////////////////////////////////////////////
// CreateSymlink
////////////////////////////////////////////////////////////////////////

type CreateSymlinkTest struct {
	fileSystemTest

	target  string
	source  string
	perms os.FileMode

	err error
}

func init() { RegisterTestSuite(&CreateSymlinkTest{}) }

func (t *CreateSymlinkTest) SetUp(i *TestInfo) {
	// Common
	t.fileSystemTest.SetUp(i)

	// Set up defaults.
	t.source = path.Join(t.baseDir, "taco")
	t.target = "/foo/bar"
	t.perms = 0644
}

func (t *CreateSymlinkTest) call() {
	t.err = t.fileSystem.CreateSymlink(t.target, t.source, t.perms)
}

func (t *CreateSymlinkTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

func (t *CreateSymlinkTest) NonExistentParent() {
	t.source = "/foo/bar/baz/qux"

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("qux")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *CreateSymlinkTest) NoPermissionsForParent() {
	dirpath := path.Join(t.baseDir, "foo")
	t.source = path.Join(dirpath, "taco")

	// Parent
	err := os.Mkdir(dirpath, 0100)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("foo")))
	ExpectThat(t.err, Error(HasSubstr("permission denied")))
}

func (t *CreateSymlinkTest) FileAlreadyExistsWithSameName() {
	// Create
	err := ioutil.WriteFile(t.source, []byte{}, 0644)
	AssertEq(nil, err)

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("file exists")))
}

func (t *CreateSymlinkTest) CreatesCorrectEntry() {
	t.source = path.Join(t.baseDir, "taco")
	t.target = "/burrito"
	t.perms = 0674  // Conflicts with default umask

	// Call
	t.call()
	AssertEq(nil, t.err)

	// List
	entries := t.list()

	AssertThat(entries, ElementsAre(Any()))
	entry := entries[0]

	ExpectEq(fs.TypeSymlink, entry.Type)
	ExpectEq("taco", entry.Name)
	ExpectEq("/burrito", entry.Target)
	ExpectEq(0674, entry.Permissions)
}

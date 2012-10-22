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
// Helpers
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

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

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

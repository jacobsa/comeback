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
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestPerms(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// SetPermissions
////////////////////////////////////////////////////////////////////////

type SetPermissionsTest struct {
	fileSystemTest

	path string
	perms os.FileMode

	err error
}

func init() { RegisterTestSuite(&SetPermissionsTest{}) }

func (t *SetPermissionsTest) call() {
	t.err = t.fileSystem.SetPermissions(t.path, t.perms)
}

func (t *SetPermissionsTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

func (t *SetPermissionsTest) NonExistentPath() {
	t.path = path.Join(t.baseDir, "foobar")

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("foobar")))
	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *SetPermissionsTest) File() {
	t.path = path.Join(t.baseDir, "taco.txt")
	t.perms = 0754

	// Create
	err := ioutil.WriteFile(t.path, []byte(""), 0600)
	AssertEq(nil, err)

	// Call
	t.call()
	AssertEq(nil, t.err)

	// List
	entries := t.list()

	AssertThat(entries, ElementsAre(Any()))
	entry := entries[0]

	AssertEq(fs.TypeFile, entry.Type)
	ExpectEq(0754, entry.Permissions)
}

func (t *SetPermissionsTest) Directory() {
	ExpectEq("TODO", "")
}

func (t *SetPermissionsTest) Symlink() {
	ExpectEq("TODO", "")
}

func (t *SetPermissionsTest) Device() {
	ExpectEq("TODO", "")
}

func (t *SetPermissionsTest) IgnoresOtherBits() {
	ExpectEq("TODO", "")
}

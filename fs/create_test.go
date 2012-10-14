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
	. "github.com/jacobsa/ogletest"
	"io"
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

	w io.WriteCloser
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
	ExpectEq("TODO", "")
}

func (t *CreateFileTest) NoPermissionsForParent() {
	ExpectEq("TODO", "")
}

func (t *CreateFileTest) FileAlreadyExists() {
	ExpectEq("TODO", "")
}

func (t *CreateFileTest) SetsPermissions() {
	ExpectEq("TODO", "")
}

func (t *CreateFileTest) SavesDataToCorrectPlace() {
	ExpectEq("TODO", "")
}

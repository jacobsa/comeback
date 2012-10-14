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
	"path"
	"testing"
	"time"
)

func TestModTime(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type SetModTimeTest struct {
	fileSystemTest

	path  string
	mtime time.Time

	err error
}

func init() { RegisterTestSuite(&SetModTimeTest{}) }

func (t *SetModTimeTest) SetUp(i *TestInfo) {
	// Common
	t.fileSystemTest.SetUp(i)

	// Set a default time.
	t.mtime = time.Date(2012, time.August, 15, 12, 56, 00, 0, time.Local)
}

func (t *SetModTimeTest) call() {
	t.err = t.fileSystem.SetModTime(t.path, t.mtime)
}

func (t *SetModTimeTest) list() []*fs.DirectoryEntry {
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	return entries
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *SetModTimeTest) NonExistentPath() {
	t.path = path.Join(t.baseDir, "foobar")

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("no such")))
}

func (t *SetModTimeTest) File() {
	t.path = path.Join(t.baseDir, "taco.txt")

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
	ExpectTrue(t.mtime.Equal(entry.MTime), "MTime: %v", entry.MTime)
}

func (t *SetModTimeTest) Directory() {
	ExpectEq("TODO", "")
}

func (t *SetModTimeTest) Symlink() {
	ExpectEq("TODO", "")
}

func (t *SetModTimeTest) NamedPipe() {
	ExpectEq("TODO", "")
}

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
	"log"
	"os"
	"path"
	"testing"
	"time"
)

func TestFileSystemTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// ReadDir
////////////////////////////////////////////////////////////////////////

type ReadDirTest struct {
	fileSystem fs.FileSystem
	baseDir string
}

func init() { RegisterTestSuite(&ReadDirTest{}) }

func (t *ReadDirTest) SetUp(i *TestInfo) {
	t.fileSystem = fs.NewFileSystem()

	// Create a temporary directory.
	var err error
	t.baseDir, err = ioutil.TempDir("", "ReadDirTest_")
	if err != nil {
		log.Fatalf("Creating baseDir: %v", err)
	}
}

func (t *ReadDirTest) TearDown() {
	err := os.RemoveAll(t.baseDir)
	if err != nil {
		log.Fatalf("Couldn't remove: %s", t.baseDir)
	}
}

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

func (t *ReadDirTest) RegularFiles() {
	var err error
	var entry *fs.DirectoryEntry

	// File 0
	path0 := path.Join(t.baseDir, "taco.txt")
	err = ioutil.WriteFile(path0, []byte("taco"), 0714 | os.ModeSetuid)
	AssertEq(nil, err)

	mtime0 := time.Date(2009, time.November, 10, 23, 0, 0, 123e6, time.UTC)
	err = os.Chtimes(path0, time.Now(), mtime0)
	AssertEq(nil, err)

	// File 1
	path1 := path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(path1, []byte("burrito"), 0464 | os.ModeSetuid | os.ModeSetgid)
	AssertEq(nil, err)

	mtime1 := time.Date(1985, time.March, 18, 15, 33, 0, 17e6, time.Local)
	err = os.Chtimes(path1, time.Now(), mtime1)
	AssertEq(nil, err)

	// File 2
	path2 := path.Join(t.baseDir, "enchilada.txt")
	err = ioutil.WriteFile(path2, []byte("enchilada"), 0111 | os.ModeSticky)
	AssertEq(nil, err)

	mtime2 := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	err = os.Chtimes(path2, time.Now(), mtime2)
	AssertEq(nil, err)

	// Call
	entries, err := t.fileSystem.ReadDir(t.baseDir)
	AssertEq(nil, err)
	AssertThat(entries, ElementsAre(Any(), Any(), Any()))

	entry = entries[0]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("burrito.txt", entry.Name)
	ExpectEq(0464 | os.ModeSetuid | os.ModeSetgid, entry.Permissions)
	ExpectTrue(entry.MTime.Equal(mtime1), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	entry = entries[1]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("enchilada.txt", entry.Name)
	ExpectEq(0111 | os.ModeSticky, entry.Permissions)
	ExpectTrue(entry.MTime.Equal(mtime2), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())

	entry = entries[2]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("taco.txt", entry.Name)
	ExpectEq(0714 | os.ModeSetuid, entry.Permissions)
	ExpectTrue(entry.MTime.Equal(mtime0), "%v", entry.MTime)
	ExpectThat(entry.Scores, ElementsAre())
}

func (t *ReadDirTest) Directories() {
	ExpectEq("TODO", "")
}

func (t *ReadDirTest) Symlinks() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// OpenForReading
////////////////////////////////////////////////////////////////////////

type OpenForReadingTest struct {
}

func init() { RegisterTestSuite(&OpenForReadingTest{}) }

func (t *OpenForReadingTest) NonExistentFile() {
	ExpectEq("TODO", "")
}

func (t *OpenForReadingTest) NotAFile() {
	ExpectEq("TODO", "")
}

func (t *OpenForReadingTest) NoReadPermissions() {
	ExpectEq("TODO", "")
}

func (t *OpenForReadingTest) EmptyFile() {
	ExpectEq("TODO", "")
}

func (t *OpenForReadingTest) FileWithContents() {
	ExpectEq("TODO", "")
}

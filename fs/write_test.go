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
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestWrite(t *testing.T) { RunTests(t) }

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

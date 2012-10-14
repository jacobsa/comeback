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
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"io/ioutil"
	"path"
	"testing"
)

func TestOpenForReading(t *testing.T) { RunTests(t) }

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

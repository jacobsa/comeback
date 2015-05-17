// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package save_test

import (
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/net/context"

	. "github.com/jacobsa/ogletest"
)

func TestFileSystemVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FileSystemVisitorTest struct {
	ctx context.Context

	// A temporary directory that is cleaned up at the end of the test.
	dir string
}

var _ SetUpInterface = &FileSystemVisitorTest{}
var _ TearDownInterface = &FileSystemVisitorTest{}

func init() { RegisterTestSuite(&FileSystemVisitorTest{}) }

func (t *FileSystemVisitorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Create the temporary directory.
	var err error
	t.dir, err = ioutil.TempDir("", "file_system_visistor_test")
	AssertEq(nil, err)
}

func (t *FileSystemVisitorTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSystemVisitorTest) NonExistentPath() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) NotADirectory() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) VisitRootNode() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) VisitNonRootNode() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Files() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Directories() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Symlinks() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Devices() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) CharDevices() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) NamedPipes() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Sockets() {
	AssertFalse(true, "TODO")
}

func (t *FileSystemVisitorTest) Exclusions() {
	AssertFalse(true, "TODO")
}

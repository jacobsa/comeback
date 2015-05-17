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
	"path"
	"sort"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/save"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestFileSystemVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type PathAndFileInfoSlice []save.PathAndFileInfo

func (p PathAndFileInfoSlice) Len() int {
	return len(p)
}

func (p PathAndFileInfoSlice) Less(i, j int) bool {
	return p[i].Path < p[j].Path
}

func (p PathAndFileInfoSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FileSystemVisitorTest struct {
	ctx context.Context

	// A temporary directory that is cleaned up at the end of the test. This is
	// the base path with which the visitor is configured.
	dir string

	// The channel into which the visitor writes. Configured with a large buffer.
	output chan save.PathAndFileInfo

	visitor graph.Visitor
}

var _ SetUpInterface = &FileSystemVisitorTest{}
var _ TearDownInterface = &FileSystemVisitorTest{}

func init() { RegisterTestSuite(&FileSystemVisitorTest{}) }

func (t *FileSystemVisitorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.output = make(chan save.PathAndFileInfo, 10e3)

	// Create the base directory.
	var err error
	t.dir, err = ioutil.TempDir("", "file_system_visistor_test")
	AssertEq(nil, err)

	// And the visitor.
	t.visitor = save.NewFileSystemVisitor(t.dir, t.output)
}

func (t *FileSystemVisitorTest) TearDown() {
	var err error

	// Clean up the junk we left in the file system.
	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

// Consume the output, returning a slice sorted by path.
func (t *FileSystemVisitorTest) sortOutput() (output PathAndFileInfoSlice) {
	close(t.output)
	for o := range t.output {
		output = append(output, o)
	}

	sort.Sort(output)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSystemVisitorTest) NonExistentPath() {
	const node = "foo/bar"

	_, err := t.visitor.Visit(t.ctx, node)
	ExpectThat(err, Error(HasSubstr(node)))
	ExpectThat(err, Error(HasSubstr("not found")))
}

func (t *FileSystemVisitorTest) NotADirectory() {
	const node = "foo"
	var err error

	// Create a file.
	err = ioutil.WriteFile(path.Join(t.dir, node), []byte{}, 0500)
	AssertEq(nil, err)

	// Attempt to visit it.
	_, err = t.visitor.Visit(t.ctx, node)
	ExpectThat(err, Error(HasSubstr(node)))
	ExpectThat(err, Error(HasSubstr("TODO")))
}

func (t *FileSystemVisitorTest) VisitRootNode() {
	var err error

	// Create two children.
	err = ioutil.WriteFile(path.Join(t.dir, "foo"), []byte{}, 0500)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.dir, "bar"), []byte{}, 0500)
	AssertEq(nil, err)

	// Visit the root.
	_, err = t.visitor.Visit(t.ctx, "")
	AssertEq(nil, err)

	// Check the output.
	output := t.sortOutput()
	AssertEq(2, len(output))
	ExpectEq(path.Join(t.dir, "bar"), output[0].Path)
	ExpectEq(path.Join(t.dir, "foo"), output[1].Path)
}

func (t *FileSystemVisitorTest) VisitNonRootNode() {
	var err error

	// Make a few levels of sub-directories.
	d := path.Join(t.dir, "sub/dirs")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create two children.
	err = ioutil.WriteFile(path.Join(d, "foo"), []byte{}, 0500)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(d, "bar"), []byte{}, 0500)
	AssertEq(nil, err)

	// Visit the root.
	_, err = t.visitor.Visit(t.ctx, "sub/dirs")
	AssertEq(nil, err)

	// Check the output.
	output := t.sortOutput()
	AssertEq(2, len(output))
	ExpectEq(path.Join(t.dir, "bar"), output[0].Path)
	ExpectEq(path.Join(t.dir, "foo"), output[1].Path)
}

func (t *FileSystemVisitorTest) Files() {
	var err error
	var pfi save.PathAndFileInfo

	// Create two children.
	err = ioutil.WriteFile(path.Join(t.dir, "foo"), []byte("taco"), 0400)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.dir, "bar"), []byte("burrito"), 0400)
	AssertEq(nil, err)

	// Visit the root.
	adjacent, err := t.visitor.Visit(t.ctx, "")

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())

	// Check the output.
	output := t.sortOutput()
	AssertEq(2, len(output))

	pfi = output[0]
	ExpectEq(path.Join(t.dir, "bar"), pfi.Path)
	ExpectEq("bar", pfi.Info.Name())
	ExpectEq(len("burrito"), pfi.Info.Size())

	pfi = output[1]
	ExpectEq(path.Join(t.dir, "foo"), pfi.Path)
	ExpectEq("foo", pfi.Info.Name())
	ExpectEq(len("taco"), pfi.Info.Size())
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

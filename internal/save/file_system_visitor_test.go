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
	"regexp"
	"sort"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	"github.com/jacobsa/comeback/internal/save"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestFileSystemVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type PathAndFileInfoSlice []*save.PathAndFileInfo

func (p PathAndFileInfoSlice) Len() int {
	return len(p)
}

func (p PathAndFileInfoSlice) Less(i, j int) bool {
	return p[i].Path < p[j].Path
}

func (p PathAndFileInfoSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func sortNodes(nodes []graph.Node) (pfis PathAndFileInfoSlice) {
	for _, n := range nodes {
		pfis = append(pfis, n.(*save.PathAndFileInfo))
	}

	sort.Sort(pfis)
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FileSystemVisitorTest struct {
	ctx context.Context

	// A temporary directory that is cleaned up at the end of the test. This is
	// the base path with which the visitor is configured.
	dir string

	// The exclusions with which to configure the visitor.
	exclusions []*regexp.Regexp

	visitor graph.SuccessorFinder
}

var _ SetUpInterface = &FileSystemVisitorTest{}
var _ TearDownInterface = &FileSystemVisitorTest{}

func init() { RegisterTestSuite(&FileSystemVisitorTest{}) }

func (t *FileSystemVisitorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Create the base directory.
	var err error
	t.dir, err = ioutil.TempDir("", "file_system_visistor_test")
	AssertEq(nil, err)

	// And the visitor.
	t.resetVisistor()
}

func (t *FileSystemVisitorTest) TearDown() {
	var err error

	// Clean up the junk we left in the file system.
	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

func (t *FileSystemVisitorTest) resetVisistor() {
	t.visitor = save.NewFileSystemVisitor(
		t.dir,
		t.exclusions)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSystemVisitorTest) NonExistentPath() {
	node := &save.PathAndFileInfo{
		Path: "foo/bar",
	}

	_, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	ExpectThat(err, Error(HasSubstr(node.Path)))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *FileSystemVisitorTest) VisitRootNode() {
	var err error

	// Create two children.
	err = ioutil.WriteFile(path.Join(t.dir, "foo"), []byte{}, 0500)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.dir, "bar"), []byte{}, 0500)
	AssertEq(nil, err)

	// Visit the root.
	node := &save.PathAndFileInfo{
		Path: "",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))
	ExpectEq("bar", pfis[0].Path)
	ExpectEq("foo", pfis[1].Path)
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

	// Visit the directory.
	node := &save.PathAndFileInfo{
		Path: "sub/dirs",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))
	ExpectEq("sub/dirs/bar", pfis[0].Path)
	ExpectEq("sub/dirs/foo", pfis[1].Path)
}

func (t *FileSystemVisitorTest) Files() {
	var err error
	var pfi *save.PathAndFileInfo

	// Make a sub-directory.
	d := path.Join(t.dir, "dir")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create two children.
	err = ioutil.WriteFile(path.Join(d, "foo"), []byte("taco"), 0400)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(d, "bar"), []byte("burrito"), 0400)
	AssertEq(nil, err)

	// Visit.
	node := &save.PathAndFileInfo{
		Path: "dir",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/bar", pfi.Path)
	ExpectEq("bar", pfi.Info.Name())
	ExpectEq(len("burrito"), pfi.Info.Size())

	pfi = pfis[1]
	ExpectEq("dir/foo", pfi.Path)
	ExpectEq("foo", pfi.Info.Name())
	ExpectEq(len("taco"), pfi.Info.Size())
}

func (t *FileSystemVisitorTest) Directories() {
	var err error
	var pfi *save.PathAndFileInfo

	// Make a sub-directory.
	d := path.Join(t.dir, "dir")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create children.
	err = os.Mkdir(path.Join(d, "foo"), 0400)
	AssertEq(nil, err)

	err = os.Mkdir(path.Join(d, "bar"), 0400)
	AssertEq(nil, err)

	// Visit.
	node := &save.PathAndFileInfo{
		Path: "dir",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/bar", pfi.Path)
	ExpectEq("bar", pfi.Info.Name())
	ExpectTrue(pfi.Info.IsDir())

	pfi = pfis[1]
	ExpectEq("dir/foo", pfi.Path)
	ExpectEq("foo", pfi.Info.Name())
	ExpectTrue(pfi.Info.IsDir())
}

func (t *FileSystemVisitorTest) Symlinks() {
	var err error
	var pfi *save.PathAndFileInfo

	// Make a sub-directory.
	d := path.Join(t.dir, "dir")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create a child.
	err = os.Symlink("blah/blah", path.Join(d, "foo"))
	AssertEq(nil, err)

	// Visit.
	node := &save.PathAndFileInfo{
		Path: "dir",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(1, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/foo", pfi.Path)
	ExpectEq("foo", pfi.Info.Name())
	ExpectFalse(pfi.Info.IsDir())
}

func (t *FileSystemVisitorTest) Exclusions() {
	var err error

	// Make a sub-directory.
	d := path.Join(t.dir, "dir")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create some children.
	err = ioutil.WriteFile(path.Join(d, "foo"), []byte{}, 0700)
	AssertEq(nil, err)

	err = os.Mkdir(path.Join(d, "bar"), 0700)
	AssertEq(nil, err)

	err = os.Symlink("blah/blah", path.Join(d, "baz"))
	AssertEq(nil, err)

	// Exclude all of them.
	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile("dir/foo"),
		regexp.MustCompile("(bar|baz)"),
	}

	t.resetVisistor()

	// Visit.
	node := &save.PathAndFileInfo{
		Path: "dir",
	}

	successors, err := t.visitor.FindDirectSuccessors(t.ctx, node)

	AssertEq(nil, err)
	ExpectThat(successors, ElementsAre())
}

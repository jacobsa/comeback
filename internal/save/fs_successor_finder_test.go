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

package save

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/graph"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestFSSuccessorFinder(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type fsNodeSlice []*fsNode

func (p fsNodeSlice) Len() int {
	return len(p)
}

func (p fsNodeSlice) Less(i, j int) bool {
	return p[i].RelPath < p[j].RelPath
}

func (p fsNodeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func sortNodes(graphNodes []graph.Node) (nodes fsNodeSlice) {
	for _, n := range graphNodes {
		nodes = append(nodes, n.(*fsNode))
	}

	sort.Sort(nodes)
	return
}

// A valid os.FileInfo for a directory.
var gDirInfo os.FileInfo

func init() {
	var err error
	gDirInfo, err = os.Stat("/")
	if err != nil {
		log.Fatalf("Stat: %v", err)
	}
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type FSSuccessorFinderTest struct {
	ctx context.Context

	// A temporary directory that is cleaned up at the end of the test. This is
	// the base path with which the successor finder is configured.
	dir string

	// The exclusions with which to configure the successor finder.
	exclusions []*regexp.Regexp

	sf graph.SuccessorFinder
}

var _ SetUpInterface = &FSSuccessorFinderTest{}
var _ TearDownInterface = &FSSuccessorFinderTest{}

func init() { RegisterTestSuite(&FSSuccessorFinderTest{}) }

func (t *FSSuccessorFinderTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx

	// Create the base directory.
	var err error
	t.dir, err = ioutil.TempDir("", "file_system_visistor_test")
	AssertEq(nil, err)

	// And the successor finder.
	t.resetVisistor()
}

func (t *FSSuccessorFinderTest) TearDown() {
	var err error

	// Clean up the junk we left in the file system.
	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

func (t *FSSuccessorFinderTest) resetVisistor() {
	t.sf = newSuccessorFinder(t.dir, t.exclusions)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FSSuccessorFinderTest) NonExistentPath() {
	node := &fsNode{
		RelPath: "foo/bar",
		Info:    gDirInfo,
	}

	_, err := t.sf.FindDirectSuccessors(t.ctx, node)
	ExpectThat(err, Error(HasSubstr(node.RelPath)))
	ExpectThat(err, Error(HasSubstr("no such file")))
}

func (t *FSSuccessorFinderTest) VisitRootNode() {
	var err error

	// Create two children.
	err = ioutil.WriteFile(path.Join(t.dir, "foo"), []byte{}, 0500)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.dir, "bar"), []byte{}, 0500)
	AssertEq(nil, err)

	// Visit the root.
	node := &fsNode{
		RelPath: "",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))
	ExpectEq("bar", pfis[0].RelPath)
	ExpectEq("foo", pfis[1].RelPath)

	// The children should have been recorded.
	ExpectThat(node.Children, ElementsAre(pfis[0], pfis[1]))
}

func (t *FSSuccessorFinderTest) VisitNonRootNode() {
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
	node := &fsNode{
		RelPath: "sub/dirs",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))
	ExpectEq("sub/dirs/bar", pfis[0].RelPath)
	ExpectEq("sub/dirs/foo", pfis[1].RelPath)

	// The children should have been recorded.
	ExpectThat(node.Children, ElementsAre(pfis[0], pfis[1]))
}

func (t *FSSuccessorFinderTest) VisitFileNode() {
	var err error

	// Create a file.
	f := path.Join(t.dir, "foo")
	err = ioutil.WriteFile(f, []byte{}, 0500)
	AssertEq(nil, err)

	fi, err := os.Stat(f)
	AssertEq(nil, err)

	// Visit it.
	node := &fsNode{
		RelPath: "foo",
		Info:    fi,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	ExpectThat(successors, ElementsAre())
	ExpectThat(node.Children, ElementsAre())
}

func (t *FSSuccessorFinderTest) Files() {
	var err error
	var pfi *fsNode

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
	node := &fsNode{
		RelPath: "dir",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/bar", pfi.RelPath)
	ExpectEq("bar", pfi.Info.Name())
	ExpectEq("", pfi.Info.Target)
	ExpectEq(len("burrito"), pfi.Info.Size())
	ExpectEq(node, pfi.Parent)

	pfi = pfis[1]
	ExpectEq("dir/foo", pfi.RelPath)
	ExpectEq("foo", pfi.Info.Name())
	ExpectEq("", pfi.Info.Target)
	ExpectEq(len("taco"), pfi.Info.Size())
	ExpectEq(node, pfi.Parent)
}

func (t *FSSuccessorFinderTest) Directories() {
	var err error
	var pfi *fsNode

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
	node := &fsNode{
		RelPath: "dir",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(2, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/bar", pfi.RelPath)
	ExpectEq("bar", pfi.Info.Name())
	ExpectEq("", pfi.Info.Target)
	ExpectTrue(pfi.Info.IsDir())
	ExpectEq(node, pfi.Parent)

	pfi = pfis[1]
	ExpectEq("dir/foo", pfi.RelPath)
	ExpectEq("foo", pfi.Info.Name())
	ExpectEq("", pfi.Info.Target)
	ExpectTrue(pfi.Info.IsDir())
	ExpectEq(node, pfi.Parent)
}

func (t *FSSuccessorFinderTest) Symlinks() {
	var err error
	var pfi *fsNode

	// Make a sub-directory.
	d := path.Join(t.dir, "dir")

	err = os.MkdirAll(d, 0700)
	AssertEq(nil, err)

	// Create a child.
	err = os.Symlink("blah/blah", path.Join(d, "foo"))
	AssertEq(nil, err)

	// Visit.
	node := &fsNode{
		RelPath: "dir",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)
	AssertEq(nil, err)

	// Check the output.
	pfis := sortNodes(successors)
	AssertEq(1, len(pfis))

	pfi = pfis[0]
	ExpectEq("dir/foo", pfi.RelPath)
	ExpectEq("foo", pfi.Info.Name())
	ExpectEq("blah/blah", pfi.Info.Target)
	ExpectFalse(pfi.Info.IsDir())
	ExpectEq(node, pfi.Parent)
}

func (t *FSSuccessorFinderTest) Exclusions() {
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
	node := &fsNode{
		RelPath: "dir",
		Info:    gDirInfo,
	}

	successors, err := t.sf.FindDirectSuccessors(t.ctx, node)

	AssertEq(nil, err)
	ExpectThat(successors, ElementsAre())
	ExpectThat(node.Children, ElementsAre())
}

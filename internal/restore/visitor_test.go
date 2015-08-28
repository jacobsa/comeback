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

package restore

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx       context.Context
	blobStore blob.Store

	// A directory that is deleted when the test completes.
	dir string

	// A visitor configured with the above directory.
	visitor dag.Visitor
}

var _ SetUpInterface = &VisitorTest{}
var _ TearDownInterface = &VisitorTest{}

func init() { RegisterTestSuite(&VisitorTest{}) }

func (t *VisitorTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Create the blob store.
	t.blobStore, err = newFakeBlobStore(t.ctx)
	AssertEq(nil, err)

	// Set up the directory.
	t.dir, err = ioutil.TempDir("", "visitor_test")
	AssertEq(nil, err)

	// Create the visitor.
	t.visitor = newVisitor(t.dir, t.blobStore, log.New(ioutil.Discard, "", 0))
}

func (t *VisitorTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

func (t *VisitorTest) call(n *node) (err error) {
	err = t.visitor.Visit(t.ctx, n)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) ParentDirsAlreadyExist() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_MissingBlob() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_CorruptBlob() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_Empty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_NonEmpty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_PermsAndModTime() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Directory_MissingBlob() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Directory_CorruptBlob() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Directory() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Symlink() {
	var err error

	n := &node{
		RelPath: "foo/bar/baz",
		Info: fs.DirectoryEntry{
			Type:        fs.TypeSymlink,
			Name:        "baz",
			Permissions: 0741,
			MTime:       time.Date(2012, time.August, 15, 12, 56, 00, 0, time.Local),
			Target:      "taco/burrito",
		},
	}

	// Call
	err = t.call(n)
	AssertEq(nil, err)

	// Stat
	p := path.Join(t.dir, n.RelPath)
	fi, err := os.Lstat(p)
	AssertEq(nil, err)

	ExpectEq("baz", fi.Name())
	ExpectEq(0741|os.ModeSymlink, fi.Mode())
	ExpectThat(fi.ModTime(), timeutil.TimeEq(n.Info.MTime))

	// Readlink
	target, err := os.Readlink(p)

	AssertEq(nil, err)
	ExpectEq("taco/burrito", target)
}

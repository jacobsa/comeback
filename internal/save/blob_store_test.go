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
	"os"
	"path"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestBlobStore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx       context.Context
	blobStore blob.Store

	node fsNode

	// A temporary directory removed at the end of the test.
	dir string
}

func init() { RegisterTestSuite(&VisitorTest{}) }

var _ SetUpInterface = &VisitorTest{}
var _ TearDownInterface = &VisitorTest{}

func (t *VisitorTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")

	// Set up the directory.
	t.dir, err = ioutil.TempDir("", "score_map_test")
	AssertEq(nil, err)
}

func (t *VisitorTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

func (t *VisitorTest) call() (err error) {
	visitor := newVisitor(t.blobStore, make(chan *fsNode, 1))
	err = visitor.Visit(t.ctx, &t.node)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) ScoresAlreadyPresent_Empty() {
	scores := []blob.Score{}
	t.node.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectThat(t.node.Scores, DeepEquals(scores))
}

func (t *VisitorTest) ScoresAlreadyPresent_NonEmpty() {
	scores := []blob.Score{blob.ComputeScore([]byte("taco"))}
	t.node.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectThat(t.node.Scores, DeepEquals(scores))
}

func (t *VisitorTest) Symlink() {
	var err error

	// Node setup
	p := path.Join(t.dir, "foo")

	err = os.Symlink("blah", p)
	AssertEq(nil, err)

	t.node.Info, err = os.Lstat(p)
	AssertEq(nil, err)
	AssertEq(os.ModeSymlink, t.node.Info.Mode()&os.ModeType)

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Scores)
}

func (t *VisitorTest) Directory() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_Empty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsFull() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsPartial() {
	AssertTrue(false, "TODO")
}

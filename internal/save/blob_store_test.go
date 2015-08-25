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
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestBlobStore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx       context.Context
	chunkSize int
	blobStore mock_blob.MockStore

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
	t.chunkSize = 8
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
	visitor := newVisitor(t.chunkSize, t.blobStore, make(chan *fsNode, 1))
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
	var err error

	// Node setup
	t.node.Info, err = os.Lstat(t.dir)
	AssertEq(nil, err)

	// Add one child.
	err = ioutil.WriteFile(path.Join(t.dir, "foo"), []byte("taco"), 0700)
	AssertEq(nil, err)

	// Snoop on the call to the blob store.
	var savedBlob []byte
	expectedScore := blob.ComputeScore([]byte("taco"))

	ExpectCall(t.blobStore, "Store")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &savedBlob), Return(expectedScore, nil)))

	// Call
	err = t.call()
	AssertEq(nil, err)
	AssertThat(t.node.Scores, ElementsAre(expectedScore))

	entries, err := repr.UnmarshalDir(savedBlob)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	entry := entries[0]
	ExpectEq("foo", entry.Name)
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq(len("taco"), entry.Size)
}

func (t *VisitorTest) File_Empty() {
	var err error

	// Node setup
	p := path.Join(t.dir, "foo")

	err = ioutil.WriteFile(p, []byte(""), 0700)
	AssertEq(nil, err)

	t.node.Info, err = os.Lstat(p)
	AssertEq(nil, err)

	// Call
	err = t.call()
	AssertEq(nil, err)

	AssertNe(nil, t.node.Scores)
	ExpectThat(t.node.Scores, ElementsAre())
}

func (t *VisitorTest) File_LastChunkIsFull() {
	var err error

	// Node setup
	p := path.Join(t.dir, "foo")

	chunk0 := bytes.Repeat([]byte{0}, t.chunkSize)
	chunk1 := bytes.Repeat([]byte{1}, t.chunkSize)

	var contents []byte
	contents = append(contents, chunk0...)
	contents = append(contents, chunk1...)

	err = ioutil.WriteFile(p, contents, 0700)
	AssertEq(nil, err)

	t.node.Info, err = os.Lstat(p)
	AssertEq(nil, err)

	// Blob store
	score0 := blob.ComputeScore(chunk0)
	ExpectCall(t.blobStore, "Store")(Any(), DeepEquals(chunk0)).
		WillOnce(Return(score0, nil))

	score1 := blob.ComputeScore(chunk0)
	ExpectCall(t.blobStore, "Store")(Any(), DeepEquals(chunk0)).
		WillOnce(Return(score1, nil))

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(t.node.Scores, ElementsAre(score0, score1))
}

func (t *VisitorTest) File_LastChunkIsPartial() {
	var err error

	// Node setup
	p := path.Join(t.dir, "foo")

	chunk0 := bytes.Repeat([]byte{0}, t.chunkSize)
	chunk1 := bytes.Repeat([]byte{1}, t.chunkSize-1)

	var contents []byte
	contents = append(contents, chunk0...)
	contents = append(contents, chunk1...)

	err = ioutil.WriteFile(p, contents, 0700)
	AssertEq(nil, err)

	t.node.Info, err = os.Lstat(p)
	AssertEq(nil, err)

	// Blob store
	score0 := blob.ComputeScore(chunk0)
	ExpectCall(t.blobStore, "Store")(Any(), DeepEquals(chunk0)).
		WillOnce(Return(score0, nil))

	score1 := blob.ComputeScore(chunk0)
	ExpectCall(t.blobStore, "Store")(Any(), DeepEquals(chunk0)).
		WillOnce(Return(score1, nil))

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(t.node.Scores, ElementsAre(score0, score1))
}

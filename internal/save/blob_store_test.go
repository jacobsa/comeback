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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"

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
// Helpers
////////////////////////////////////////////////////////////////////////

// Match *blob.StoreRequest where the Blob field's contents are equal to the
// given slice.
func blobEquals(expected []byte) Matcher {
	pred := func(c interface{}) (err error) {
		req, ok := c.(*blob.StoreRequest)
		if !ok {
			err = fmt.Errorf("which has type %T", c)
			return
		}

		if !bytes.Equal(expected, req.Blob) {
			err = errors.New("which has different contents")
			return
		}

		return
	}

	return NewMatcher(pred, "*blob.StoreRequest with appropriate content")
}

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
	visitor := newVisitor(
		t.chunkSize,
		t.dir,
		t.blobStore,
		log.New(ioutil.Discard, "", 0),
		make(chan *fsNode, 1))

	err = visitor.Visit(t.ctx, &t.node)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) ScoresAlreadyPresent_Empty() {
	scores := []blob.Score{}
	t.node.Info.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectThat(t.node.Info.Scores, DeepEquals(scores))
}

func (t *VisitorTest) ScoresAlreadyPresent_NonEmpty() {
	scores := []blob.Score{blob.ComputeScore([]byte("taco"))}
	t.node.Info.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectThat(t.node.Info.Scores, DeepEquals(scores))
}

func (t *VisitorTest) Symlink() {
	var err error

	// Node setup
	t.node.RelPath = "foo"
	t.node.Info = fs.DirectoryEntry{
		Type: fs.TypeSymlink,
	}

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Info.Scores)
}

func (t *VisitorTest) Directory() {
	var err error

	// Children
	child0 := &fsNode{
		Info: fs.DirectoryEntry{
			Name:  "taco",
			MTime: time.Date(2012, time.August, 15, 12, 56, 00, 0, time.Local),
		},
	}

	child1 := &fsNode{
		Info: fs.DirectoryEntry{
			Name:  "burrito",
			MTime: time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local),
		},
	}

	// Node setup
	t.node.RelPath = ""
	t.node.Info = fs.DirectoryEntry{
		Type: fs.TypeDirectory,
	}

	t.node.Children = []*fsNode{child0, child1}

	// Snoop on the call to the blob store.
	var savedReq *blob.StoreRequest
	expectedScore := blob.ComputeScore([]byte("taco"))

	ExpectCall(t.blobStore, "Store")(Any(), Any()).
		WillOnce(DoAll(SaveArg(1, &savedReq), Return(expectedScore, nil)))

	// Call
	err = t.call()
	AssertEq(nil, err)
	AssertThat(t.node.Info.Scores, ElementsAre(expectedScore))

	// Parse the blob.
	entries, err := repr.UnmarshalDir(savedReq.Blob)
	AssertEq(nil, err)
	AssertEq(2, len(entries))

	ExpectThat(*entries[0], DeepEquals(child0.Info))
	ExpectThat(*entries[1], DeepEquals(child1.Info))
}

func (t *VisitorTest) File_Empty() {
	var err error

	// Node setup
	t.node.RelPath = "foo"
	t.node.Info.Type = fs.TypeFile
	p := path.Join(t.dir, t.node.RelPath)

	err = ioutil.WriteFile(p, []byte(""), 0700)
	AssertEq(nil, err)

	// Call
	err = t.call()
	AssertEq(nil, err)

	AssertNe(nil, t.node.Info.Scores)
	ExpectThat(t.node.Info.Scores, ElementsAre())
}

func (t *VisitorTest) File_LastChunkIsFull() {
	var err error

	// Node setup
	t.node.RelPath = "foo"
	t.node.Info.Type = fs.TypeFile
	p := path.Join(t.dir, t.node.RelPath)

	chunk0 := bytes.Repeat([]byte{0}, t.chunkSize)
	chunk1 := bytes.Repeat([]byte{1}, t.chunkSize)

	var contents []byte
	contents = append(contents, chunk0...)
	contents = append(contents, chunk1...)

	err = ioutil.WriteFile(p, contents, 0700)
	AssertEq(nil, err)

	// Blob store (chunk 0)
	expected0, err := repr.MarshalFile(chunk0)
	AssertEq(nil, err)

	score0 := blob.ComputeScore(expected0)
	ExpectCall(t.blobStore, "Store")(Any(), blobEquals(expected0)).
		WillOnce(Return(score0, nil))

	// Blob store (chunk 1)
	expected1, err := repr.MarshalFile(chunk1)
	AssertEq(nil, err)

	score1 := blob.ComputeScore(expected1)
	ExpectCall(t.blobStore, "Store")(Any(), blobEquals(expected1)).
		WillOnce(Return(score1, nil))

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(t.node.Info.Scores, ElementsAre(score0, score1))
}

func (t *VisitorTest) File_LastChunkIsPartial() {
	var err error

	// Node setup
	t.node.RelPath = "foo"
	p := path.Join(t.dir, t.node.RelPath)
	t.node.Info.Type = fs.TypeFile

	chunk0 := bytes.Repeat([]byte{0}, t.chunkSize)
	chunk1 := bytes.Repeat([]byte{1}, t.chunkSize-1)

	var contents []byte
	contents = append(contents, chunk0...)
	contents = append(contents, chunk1...)

	err = ioutil.WriteFile(p, contents, 0700)
	AssertEq(nil, err)

	// Blob store (chunk 0)
	expected0, err := repr.MarshalFile(chunk0)
	AssertEq(nil, err)

	score0 := blob.ComputeScore(expected0)
	ExpectCall(t.blobStore, "Store")(Any(), blobEquals(expected0)).
		WillOnce(Return(score0, nil))

	// Blob store (chunk 1)
	expected1, err := repr.MarshalFile(chunk1)
	AssertEq(nil, err)

	score1 := blob.ComputeScore(expected1)
	ExpectCall(t.blobStore, "Store")(Any(), blobEquals(expected1)).
		WillOnce(Return(score1, nil))

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(t.node.Info.Scores, ElementsAre(score0, score1))
}

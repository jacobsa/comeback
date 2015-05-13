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

package verify_test

import (
	"errors"
	"fmt"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/verify"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const namePrefix = "blobs/"

type superTest struct {
	ctx       context.Context
	blobStore mock_blob.MockStore
	visitor   graph.Visitor
}

func (t *superTest) setUp(
	ti *TestInfo,
	readFiles bool,
	allScores []blob.Score) {
	t.ctx = ti.Ctx
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")
	t.visitor = verify.NewVisitor(readFiles, allScores, t.blobStore)
}

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

type CommonTest struct {
	superTest
}

var _ SetUpInterface = &CommonTest{}

func init() { RegisterTestSuite(&CommonTest{}) }

func (t *CommonTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false, nil)
}

func (t *CommonTest) InvalidNodeName() {
	_, err := t.visitor.Visit(t.ctx, "taco")

	ExpectThat(err, Error(HasSubstr("ParseNodeName")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

////////////////////////////////////////////////////////////////////////
// Directories
////////////////////////////////////////////////////////////////////////

type DirsTest struct {
	superTest

	listing  []*fs.DirectoryEntry
	contents []byte
	score    blob.Score
	node     string
}

var _ SetUpInterface = &DirsTest{}

func init() { RegisterTestSuite(&DirsTest{}) }

func (t *DirsTest) SetUp(ti *TestInfo) {
	t.superTest.setUp(ti, false, nil)

	// Set up canned data for a valid listing.
	t.listing = []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type: fs.TypeFile,
			Name: "foo",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("0")),
				blob.ComputeScore([]byte("1")),
			},
		},

		&fs.DirectoryEntry{
			Type: fs.TypeDirectory,
			Name: "bar",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("2")),
			},
		},

		&fs.DirectoryEntry{
			Type:   fs.TypeSymlink,
			Name:   "baz",
			Target: "asdf",
		},
	}

	var err error
	t.contents, err = repr.MarshalDir(t.listing)
	AssertEq(nil, err)

	t.score = blob.ComputeScore(t.contents)
	t.node = verify.FormatNodeName(true, t.score)
}

func (t *DirsTest) CallsBlobStore() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), t.score).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.visitor.Visit(t.ctx, t.node)
}

func (t *DirsTest) BlobStoreReturnsError() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.visitor.Visit(t.ctx, t.node)

	ExpectThat(err, Error(HasSubstr("Load")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *DirsTest) BlobIsJunk() {
	// Set up junk contents.
	t.contents = append(t.contents, 'a')
	t.score = blob.ComputeScore(t.contents)
	t.node = verify.FormatNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err := t.visitor.Visit(t.ctx, t.node)

	ExpectThat(err, Error(HasSubstr(t.score.Hex())))
	ExpectThat(err, Error(HasSubstr("UnmarshalDir")))
}

func (t *DirsTest) UnknownEntryType() {
	// Set up a listing with an unsupported entry type.
	t.listing = []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type: fs.TypeCharDevice,
			Name: "foo",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("0")),
			},
		},
	}

	var err error
	t.contents, err = repr.MarshalDir(t.listing)
	AssertEq(nil, err)

	t.score = blob.ComputeScore(t.contents)
	t.node = verify.FormatNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.visitor.Visit(t.ctx, t.node)

	ExpectThat(err, Error(HasSubstr("entry type")))
	ExpectThat(err, Error(HasSubstr(fmt.Sprintf("%d", fs.TypeCharDevice))))
}

func (t *DirsTest) SymlinkWithScores() {
	// Set up a listing with a symlink that unexpectedly has associated scores.
	t.listing = []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type: fs.TypeSymlink,
			Name: "foo",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("0")),
			},
		},
	}

	var err error
	t.contents, err = repr.MarshalDir(t.listing)
	AssertEq(nil, err)

	t.score = blob.ComputeScore(t.contents)
	t.node = verify.FormatNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.visitor.Visit(t.ctx, t.node)

	ExpectThat(err, Error(HasSubstr(t.score.Hex())))
	ExpectThat(err, Error(HasSubstr("symlink")))
	ExpectThat(err, Error(HasSubstr("scores")))
}

func (t *DirsTest) ReturnsAppropriateAdjacentNodes() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	adjacent, err := t.visitor.Visit(t.ctx, t.node)
	AssertEq(nil, err)

	var expected []interface{}
	for _, entry := range t.listing {
		dir := entry.Type == fs.TypeDirectory
		for _, score := range entry.Scores {
			expected = append(expected, verify.FormatNodeName(dir, score))
		}
	}

	ExpectThat(adjacent, ElementsAre(expected...))
}

////////////////////////////////////////////////////////////////////////
// Files, no read
////////////////////////////////////////////////////////////////////////

type filesCommonTest struct {
	superTest

	contents []byte

	knownScore blob.Score
	knownNode  string

	unknownScore blob.Score
	unknownNode  string
}

func (t *filesCommonTest) setUp(ti *TestInfo, readFiles bool) {
	t.contents = []byte("foobarbaz")

	t.knownScore = blob.ComputeScore(t.contents)
	t.knownNode = verify.FormatNodeName(false, t.knownScore)

	t.unknownScore = blob.ComputeScore(append(t.contents, 'a'))
	t.unknownNode = verify.FormatNodeName(false, t.unknownScore)

	t.superTest.setUp(ti, readFiles, []blob.Score{t.knownScore})
}

type FilesLiteTest struct {
	filesCommonTest
}

var _ SetUpInterface = &FilesLiteTest{}

func init() { RegisterTestSuite(&FilesLiteTest{}) }

func (t *FilesLiteTest) SetUp(ti *TestInfo) {
	t.filesCommonTest.setUp(ti, false)
}

func (t *FilesLiteTest) ScoreNotInList() {
	// Call
	_, err := t.visitor.Visit(t.ctx, t.unknownNode)

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownScore.Hex())))
}

func (t *FilesLiteTest) ScoreIsInList() {
	// Call
	adjacent, err := t.visitor.Visit(t.ctx, t.knownNode)

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
}

////////////////////////////////////////////////////////////////////////
// Files, read
////////////////////////////////////////////////////////////////////////

type FilesFullTest struct {
	filesCommonTest
}

var _ SetUpInterface = &FilesFullTest{}

func init() { RegisterTestSuite(&FilesFullTest{}) }

func (t *FilesFullTest) SetUp(ti *TestInfo) {
	t.filesCommonTest.setUp(ti, true)
}

func (t *FilesFullTest) CallsBlobStore() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), t.knownScore).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.visitor.Visit(t.ctx, t.knownNode)
}

func (t *FilesFullTest) BlobStoreReturnsError() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.visitor.Visit(t.ctx, t.knownNode)

	ExpectThat(err, Error(HasSubstr("Load")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *FilesFullTest) BlobStoreSucceeds() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	adjacent, err := t.visitor.Visit(t.ctx, t.knownNode)

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
}

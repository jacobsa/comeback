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
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/comeback/internal/verify"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeNodeName(
	dir bool,
	score blob.Score) (s string) {
	n := verify.Node{
		Dir:   dir,
		Score: score,
	}

	s = n.String()
	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const namePrefix = "blobs/"

type superTest struct {
	ctx            context.Context
	readFiles      bool
	records        chan verify.Record
	allScores      []blob.Score
	knownStructure map[verify.Node][]verify.Node
	blobStore      mock_blob.MockStore
	clock          timeutil.SimulatedClock
}

func (t *superTest) setUp(
	ti *TestInfo,
	readFiles bool) {
	t.ctx = ti.Ctx
	t.readFiles = readFiles
	t.records = make(chan verify.Record, 1000)
	t.knownStructure = make(map[verify.Node][]verify.Node)
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *superTest) visit(n string) (adjacent []string, err error) {
	// Create the visitor.
	visitor := verify.NewVisitor(
		t.readFiles,
		t.allScores,
		t.knownStructure,
		t.records,
		&t.clock,
		t.blobStore)

	// Visit.
	adjacent, err = visitor.Visit(t.ctx, n)

	return
}

func (t *superTest) getRecords() (records []verify.Record) {
	close(t.records)
	for r := range t.records {
		records = append(records, r)
	}

	return
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
	t.superTest.setUp(ti, false)
}

func (t *CommonTest) InvalidNodeName() {
	_, err := t.visit("taco")

	ExpectThat(err, Error(HasSubstr("ParseNode")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

////////////////////////////////////////////////////////////////////////
// Directories
////////////////////////////////////////////////////////////////////////

type DirsTest struct {
	superTest

	knownNode   verify.Node
	unknownNode verify.Node

	listing  []*fs.DirectoryEntry
	contents []byte
	score    blob.Score
	node     string
}

var _ SetUpInterface = &DirsTest{}

func init() { RegisterTestSuite(&DirsTest{}) }

func (t *DirsTest) SetUp(ti *TestInfo) {
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
	t.node = makeNodeName(true, t.score)
	t.allScores = append(t.allScores, t.score)

	// Set up canned nodes.
	t.knownNode = verify.Node{
		Dir:   true,
		Score: blob.ComputeScore([]byte("knownNode")),
	}

	t.unknownNode = verify.Node{
		Dir:   true,
		Score: blob.ComputeScore([]byte("unknownNode")),
	}

	t.allScores = append(t.allScores, t.knownNode.Score)

	// Call through.
	t.superTest.setUp(ti, false)
}

func (t *DirsTest) NodeVisitedOnPastRun_ScoreAbsent() {
	// Set up known children for the node whose score is not in allScores.
	t.knownStructure[t.unknownNode] = []verify.Node{t.knownNode}

	// We should receive an error, and no records.
	_, err := t.visit(t.unknownNode.String())

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) NodeVisitedOnPastRun_ScorePresent() {
	// Set up known children for the node whose score is in allScores.
	t.knownStructure[t.knownNode] = []verify.Node{t.unknownNode, t.knownNode}

	// We should succeed without doing anything further. No new record should be
	// minted.
	adjacent, err := t.visit(t.knownNode.String())

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre(t.unknownNode.String(), t.knownNode.String()))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) CallsBlobStore() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), t.score).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.visit(t.node)
}

func (t *DirsTest) BlobStoreReturnsError() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.visit(t.node)

	ExpectThat(err, Error(HasSubstr("Load")))
	ExpectThat(err, Error(HasSubstr("taco")))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) BlobIsJunk() {
	// Set up junk contents.
	t.contents = append(t.contents, 'a')
	t.score = blob.ComputeScore(t.contents)
	t.allScores = append(t.allScores, t.score)
	t.node = makeNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err := t.visit(t.node)

	ExpectThat(err, Error(HasSubstr(t.score.Hex())))
	ExpectThat(err, Error(HasSubstr("UnmarshalDir")))
	ExpectThat(t.getRecords(), ElementsAre())
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
	t.allScores = append(t.allScores, t.score)
	t.node = makeNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.visit(t.node)

	ExpectThat(err, Error(HasSubstr("entry type")))
	ExpectThat(err, Error(HasSubstr(fmt.Sprintf("%d", fs.TypeCharDevice))))
	ExpectThat(t.getRecords(), ElementsAre())
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
	t.allScores = append(t.allScores, t.score)
	t.node = makeNodeName(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.visit(t.node)

	ExpectThat(err, Error(HasSubstr(t.score.Hex())))
	ExpectThat(err, Error(HasSubstr("symlink")))
	ExpectThat(err, Error(HasSubstr("scores")))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) ReturnsAppropriateAdjacentNodesAndRecords() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	adjacent, err := t.visit(t.node)
	AssertEq(nil, err)

	var expected []interface{}
	for _, entry := range t.listing {
		dir := entry.Type == fs.TypeDirectory
		for _, score := range entry.Scores {
			expected = append(expected, makeNodeName(dir, score))
		}
	}

	// Check adjacent.
	ExpectThat(adjacent, ElementsAre(expected...))

	// Check records.
	records := t.getRecords()
	AssertEq(1, len(records))
	var r verify.Record

	r = records[0]
	ExpectThat(r.Time, timeutil.TimeEq(t.clock.Now()))
	ExpectTrue(r.Node.Dir)
	ExpectEq(t.score, r.Node.Score)

	var childNames []string
	for _, child := range r.Children {
		childNames = append(childNames, child.String())
	}

	ExpectThat(childNames, ElementsAre(expected...))
}

////////////////////////////////////////////////////////////////////////
// Files, no read
////////////////////////////////////////////////////////////////////////

type filesCommonTest struct {
	superTest

	contents []byte

	knownNode   verify.Node
	unknownNode verify.Node
}

func (t *filesCommonTest) setUp(ti *TestInfo, readFiles bool) {
	t.contents = []byte("foobarbaz")

	// Set up canned nodes.
	t.knownNode = verify.Node{
		Dir:   false,
		Score: blob.ComputeScore(t.contents),
	}

	t.unknownNode = verify.Node{
		Dir:   false,
		Score: blob.ComputeScore(append(t.contents, 'a')),
	}

	t.allScores = append(t.allScores, t.knownNode.Score)

	t.superTest.setUp(ti, readFiles)
}

type FilesLiteTest struct {
	filesCommonTest
}

var _ SetUpInterface = &FilesLiteTest{}

func init() { RegisterTestSuite(&FilesLiteTest{}) }

func (t *FilesLiteTest) SetUp(ti *TestInfo) {
	t.filesCommonTest.setUp(ti, false)
}

func (t *FilesLiteTest) NodeVisitedOnPastRun_ScoreAbsent() {
	// Set up known children for the node whose score is not in allScores.
	t.knownStructure[t.unknownNode] = []verify.Node{}

	// We should receive an error, and no records.
	_, err := t.visit(t.unknownNode.String())

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesLiteTest) NodeVisitedOnPastRun_ScorePresent() {
	// Set up known children for the node whose score is in allScores.
	t.knownStructure[t.knownNode] = []verify.Node{}

	// We should succeed without doing anything further. No new record should be
	// minted.
	adjacent, err := t.visit(t.knownNode.String())

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesLiteTest) ScoreNotInList() {
	// Call
	_, err := t.visit(t.unknownNode.String())

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesLiteTest) ScoreIsInList() {
	// Call
	adjacent, err := t.visit(t.knownNode.String())

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
	ExpectThat(t.getRecords(), ElementsAre())
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

func (t *FilesFullTest) NodeVisitedOnPastRun_ScoreAbsent() {
	// Set up known children for the node whose score is not in allScores.
	t.knownStructure[t.unknownNode] = []verify.Node{}

	// We should receive an error, and no records.
	_, err := t.visit(t.unknownNode.String())

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesFullTest) NodeVisitedOnPastRun_ScorePresent() {
	// Set up known children for the node whose score is in allScores.
	t.knownStructure[t.knownNode] = []verify.Node{}

	// We should succeed without doing anything further. No new record should be
	// minted.
	adjacent, err := t.visit(t.knownNode.String())

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesFullTest) CallsBlobStore() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), t.knownNode.Score).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.visit(t.knownNode.String())
}

func (t *FilesFullTest) BlobStoreReturnsError() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.visit(t.knownNode.String())

	ExpectThat(err, Error(HasSubstr("Load")))
	ExpectThat(err, Error(HasSubstr("taco")))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesFullTest) BlobStoreSucceeds() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	adjacent, err := t.visit(t.knownNode.String())

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())

	records := t.getRecords()
	AssertEq(1, len(records))

	r := records[0]
	ExpectThat(r.Time, timeutil.TimeEq(t.clock.Now()))
	ExpectFalse(r.Node.Dir)
	ExpectEq(t.knownNode.Score, r.Node.Score)
	ExpectThat(r.Children, ElementsAre())
}
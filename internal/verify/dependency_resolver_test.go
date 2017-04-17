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

package verify

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestDependencyResolver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeNode(
	dir bool,
	score blob.Score) (n Node) {
	n = Node{
		Dir:   dir,
		Score: score,
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

const namePrefix = "blobs/"

type superTest struct {
	ctx            context.Context
	records        chan Record
	allScores      []blob.Score
	knownStructure map[Node][]Node
	blobStore      mock_blob.MockStore
	clock          timeutil.SimulatedClock
}

func (t *superTest) setUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.records = make(chan Record, 1000)
	t.knownStructure = make(map[Node][]Node)
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")
	t.clock.SetTime(time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local))
}

func (t *superTest) call(n Node) (children []string, err error) {
	dr := newDependencyResolver(
		t.allScores,
		t.knownStructure,
		t.records,
		t.blobStore,
		&t.clock)

	deps, err := dr.FindDependencies(t.ctx, n)
	for _, untyped := range deps {
		tmp := untyped.(Node)
		children = append(children, tmp.String())
	}

	return
}

func (t *superTest) getRecords() (records []Record) {
	close(t.records)
	for r := range t.records {
		records = append(records, r)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Directories
////////////////////////////////////////////////////////////////////////

type DirsTest struct {
	superTest

	knownNode   Node
	unknownNode Node

	listing  []*fs.FileInfo
	contents []byte
	score    blob.Score
	node     Node
}

var _ SetUpInterface = &DirsTest{}

func init() { RegisterTestSuite(&DirsTest{}) }

func (t *DirsTest) SetUp(ti *TestInfo) {
	// Set up canned data for a valid listing.
	t.listing = []*fs.FileInfo{
		&fs.FileInfo{
			Type: fs.TypeFile,
			Name: "foo",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("0")),
				blob.ComputeScore([]byte("1")),
			},
		},

		&fs.FileInfo{
			Type: fs.TypeDirectory,
			Name: "bar",
			Scores: []blob.Score{
				blob.ComputeScore([]byte("2")),
			},
		},

		&fs.FileInfo{
			Type:   fs.TypeSymlink,
			Name:   "baz",
			Target: "asdf",
		},
	}

	var err error
	t.contents, err = repr.MarshalDir(t.listing)
	AssertEq(nil, err)

	t.score = blob.ComputeScore(t.contents)
	t.node = makeNode(true, t.score)
	t.allScores = append(t.allScores, t.score)

	// Set up canned nodes.
	t.knownNode = Node{
		Dir:   true,
		Score: blob.ComputeScore([]byte("knownNode")),
	}

	t.unknownNode = Node{
		Dir:   true,
		Score: blob.ComputeScore([]byte("unknownNode")),
	}

	t.allScores = append(t.allScores, t.knownNode.Score)

	// Call through.
	t.superTest.setUp(ti)
}

func (t *DirsTest) NodeVisitedOnPastRun_ScoreAbsent() {
	// Set up known children for the node whose score is not in allScores.
	t.knownStructure[t.unknownNode] = []Node{t.knownNode}

	// We should receive an error, and no records.
	_, err := t.call(t.unknownNode)

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) NodeVisitedOnPastRun_ScorePresent() {
	// Set up known children for the node whose score is in allScores.
	t.knownStructure[t.knownNode] = []Node{t.unknownNode, t.knownNode}

	// We should succeed without doing anything further. No new record should be
	// minted.
	adjacent, err := t.call(t.knownNode)

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre(t.unknownNode.String(), t.knownNode.String()))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) CallsBlobStore() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), t.score).
		WillOnce(Return(nil, errors.New("")))

	// Call
	t.call(t.node)
}

func (t *DirsTest) BlobStoreReturnsError() {
	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(nil, errors.New("taco")))

	// Call
	_, err := t.call(t.node)

	ExpectThat(err, Error(HasSubstr("Load")))
	ExpectThat(err, Error(HasSubstr("taco")))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) BlobIsJunk() {
	// Set up junk contents.
	t.contents = append(t.contents, 'a')
	t.score = blob.ComputeScore(t.contents)
	t.allScores = append(t.allScores, t.score)
	t.node = makeNode(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err := t.call(t.node)

	ExpectThat(err, Error(HasSubstr(t.score.Hex())))
	ExpectThat(err, Error(HasSubstr("UnmarshalDir")))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) UnknownEntryType() {
	// Set up a listing with an unsupported entry type.
	t.listing = []*fs.FileInfo{
		&fs.FileInfo{
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
	t.node = makeNode(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.call(t.node)

	ExpectThat(err, Error(HasSubstr("entry type")))
	ExpectThat(err, Error(HasSubstr(fmt.Sprintf("%d", fs.TypeCharDevice))))
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *DirsTest) SymlinkWithScores() {
	// Set up a listing with a symlink that unexpectedly has associated scores.
	t.listing = []*fs.FileInfo{
		&fs.FileInfo{
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
	t.node = makeNode(true, t.score)

	// Load
	ExpectCall(t.blobStore, "Load")(Any(), Any()).
		WillOnce(Return(t.contents, nil))

	// Call
	_, err = t.call(t.node)

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
	adjacent, err := t.call(t.node)
	AssertEq(nil, err)

	var expected []interface{}
	for _, entry := range t.listing {
		dir := entry.Type == fs.TypeDirectory
		for _, score := range entry.Scores {
			tmp := makeNode(dir, score)
			expected = append(expected, tmp.String())
		}
	}

	// Check adjacent.
	ExpectThat(adjacent, ElementsAre(expected...))

	// Check records.
	records := t.getRecords()
	AssertEq(1, len(records))
	var r Record

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
// Files
////////////////////////////////////////////////////////////////////////

type FilesTest struct {
	superTest

	contents []byte

	knownNode   Node
	unknownNode Node
}

var _ SetUpInterface = &FilesTest{}

func init() { RegisterTestSuite(&FilesTest{}) }

func (t *FilesTest) SetUp(ti *TestInfo) {
	t.contents = []byte("foobarbaz")

	// Set up canned nodes.
	t.knownNode = Node{
		Dir:   false,
		Score: blob.ComputeScore(t.contents),
	}

	t.unknownNode = Node{
		Dir:   false,
		Score: blob.ComputeScore(append(t.contents, 'a')),
	}

	t.allScores = append(t.allScores, t.knownNode.Score)

	t.superTest.setUp(ti)
}

func (t *FilesTest) NodeVisitedOnPastRun_ScoreAbsent() {
	// Set up known children for the node whose score is not in allScores.
	t.knownStructure[t.unknownNode] = []Node{}

	// We should receive an error, and no records.
	_, err := t.call(t.unknownNode)

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesTest) NodeVisitedOnPastRun_ScorePresent() {
	// Set up known children for the node whose score is in allScores.
	t.knownStructure[t.knownNode] = []Node{}

	// We should succeed without doing anything further. No new record should be
	// minted.
	adjacent, err := t.call(t.knownNode)

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesTest) ScoreNotInList() {
	// Call
	_, err := t.call(t.unknownNode)

	ExpectThat(err, Error(HasSubstr("Unknown")))
	ExpectThat(err, Error(HasSubstr("score")))
	ExpectThat(err, Error(HasSubstr(t.unknownNode.Score.Hex())))

	ExpectThat(t.getRecords(), ElementsAre())
}

func (t *FilesTest) ScoreIsInList() {
	// Call
	adjacent, err := t.call(t.knownNode)

	AssertEq(nil, err)
	ExpectThat(adjacent, ElementsAre())
	ExpectThat(t.getRecords(), ElementsAre())
}

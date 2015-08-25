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
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/state"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestScoreMap(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type scoreMapTest struct {
	ctx      context.Context
	scoreMap state.ScoreMap
	clock    timeutil.SimulatedClock

	node fsNode
}

var _ SetUpInterface = &scoreMapTest{}

func (t *scoreMapTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.scoreMap = state.NewScoreMap()
	t.clock.SetTime(time.Date(2012, time.August, 15, 12, 56, 00, 0, time.Local))
}

////////////////////////////////////////////////////////////////////////
// makeScoreMapKey
////////////////////////////////////////////////////////////////////////

type MakeScoreMapKeyTest struct {
	scoreMapTest
}

func init() { RegisterTestSuite(&MakeScoreMapKeyTest{}) }

func (t *MakeScoreMapKeyTest) call() (key *state.ScoreMapKey) {
	key = makeScoreMapKey(&t.node, &t.clock)
	return
}

func (t *MakeScoreMapKeyTest) Directory() {
	t.node.Info.Type = fs.TypeDirectory

	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Symlink() {
	// Set up
	t.node.RelPath = "foo"
	t.node.Info = fs.DirectoryEntry{
		Type: fs.TypeSymlink,
	}

	// Call
	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) RecentlyModified() {
	var key *state.ScoreMapKey

	// Set up
	t.node.RelPath = "foo"
	t.node.Info = fs.DirectoryEntry{
		Type: fs.TypeFile,
	}

	// A short while ago
	t.node.Info.MTime = t.clock.Now().Add(-10 * time.Second)
	key = t.call()
	ExpectEq(nil, key)

	// Now
	t.node.Info.MTime = t.clock.Now()
	key = t.call()
	ExpectEq(nil, key)

	// A short while in the future
	t.node.Info.MTime = t.clock.Now().Add(10 * time.Second)
	key = t.call()
	ExpectEq(nil, key)

	// Far in the future
	t.node.Info.MTime = t.clock.Now().Add(365 * 24 * time.Hour)
	key = t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Valid() {
	// Set up
	t.node.RelPath = "foo"
	t.node.Info = fs.DirectoryEntry{
		Permissions: 0745,
		Uid:         17,
		Gid:         19,
		MTime:       t.clock.Now().Add(-10 * time.Hour),
		Inode:       23,
		Size:        31,
	}

	// Call
	key := t.call()
	AssertNe(nil, key)

	ExpectEq(t.node.RelPath, key.Path)
	ExpectEq(os.FileMode(0745), key.Permissions)
	ExpectEq(17, key.Uid)
	ExpectEq(19, key.Gid)
	ExpectThat(key.MTime, timeutil.TimeEq(t.node.Info.MTime))
	ExpectEq(23, key.Inode)
	ExpectEq(31, key.Size)
}

////////////////////////////////////////////////////////////////////////
// consultScoreMap
////////////////////////////////////////////////////////////////////////

type ConsultScoreMapTest struct {
	scoreMapTest
	expectedKey *state.ScoreMapKey
}

func init() { RegisterTestSuite(&ConsultScoreMapTest{}) }

func (t *ConsultScoreMapTest) SetUp(ti *TestInfo) {
	t.scoreMapTest.SetUp(ti)

	// Make sure the node is eligible by default.
	t.node.RelPath = "foo"
	t.node.Info = fs.DirectoryEntry{
		Type:  fs.TypeFile,
		MTime: t.clock.Now().Add(-10 * time.Hour),
	}

	t.expectedKey = makeScoreMapKey(&t.node, &t.clock)
	AssertNe(nil, t.expectedKey)
}

func (t *ConsultScoreMapTest) call() (err error) {
	nodesIn := make(chan *fsNode, 1)
	nodesIn <- &t.node
	close(nodesIn)

	err = consultScoreMap(
		t.ctx,
		t.scoreMap,
		&t.clock,
		nodesIn,
		make(chan *fsNode, 1)) // Ignore output

	return
}

func (t *ConsultScoreMapTest) NodeNotEligible() {
	var err error

	// Make the node appear as a directory.
	t.node.Info.Type = fs.TypeDirectory

	// Call. Nothing should be changed about the Info.Scores field.
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Info.Scores)
	ExpectEq(nil, t.node.ScoreMapKey)
}

func (t *ConsultScoreMapTest) PresentInScoreMap() {
	var err error

	// Prepare score map
	score1 := blob.ComputeScore([]byte("taco"))
	score2 := blob.ComputeScore([]byte("burrito"))

	t.scoreMap.Set(*t.expectedKey, []blob.Score{score1, score2})

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(t.node.Info.Scores, ElementsAre(score1, score2))
	ExpectEq(nil, t.node.ScoreMapKey)
}

func (t *ConsultScoreMapTest) AbsentInScoreMap() {
	var err error

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Info.Scores)
	ExpectThat(t.node.ScoreMapKey, Pointee(DeepEquals(*t.expectedKey)))
}

////////////////////////////////////////////////////////////////////////
// updateScoreMap
////////////////////////////////////////////////////////////////////////

type UpdateScoreMapTest struct {
	scoreMapTest
}

func init() { RegisterTestSuite(&UpdateScoreMapTest{}) }

func (t *UpdateScoreMapTest) call() (err error) {
	nodesIn := make(chan *fsNode, 1)
	nodesIn <- &t.node
	close(nodesIn)

	err = updateScoreMap(t.ctx, t.scoreMap, nodesIn)
	return
}

func (t *UpdateScoreMapTest) ScoreMapKeyMissing() {
	var err error

	// Prepare
	t.node.Info.Scores = []blob.Score{
		blob.ComputeScore([]byte("taco")),
		blob.ComputeScore([]byte("burrito")),
	}

	t.node.ScoreMapKey = nil

	// Call
	err = t.call()
	AssertEq(nil, err)
}

func (t *UpdateScoreMapTest) ScoreMapKeyPresent() {
	var err error

	// Prepare
	t.node.Info.Scores = []blob.Score{
		blob.ComputeScore([]byte("taco")),
		blob.ComputeScore([]byte("burrito")),
	}

	t.node.ScoreMapKey = &state.ScoreMapKey{
		Uid: 17,
	}

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(
		t.scoreMap.Get(*t.node.ScoreMapKey),
		ElementsAre(t.node.Info.Scores[0], t.node.Info.Scores[1]))
}

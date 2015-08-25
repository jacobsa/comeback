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
	"syscall"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
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

	// A temporary directory removed at the end of the test.
	dir string
}

var _ SetUpInterface = &scoreMapTest{}
var _ TearDownInterface = &scoreMapTest{}

func (t *scoreMapTest) SetUp(ti *TestInfo) {
	var err error

	t.ctx = ti.Ctx
	t.scoreMap = state.NewScoreMap()

	// Set up the clock with a default time far in the future, so that recent
	// modifications in the file system appear old.
	t.clock.SetTime(time.Now().Add(365 * 24 * time.Hour))

	// Set up the directory.
	t.dir, err = ioutil.TempDir("", "score_map_test")
	AssertEq(nil, err)
}

func (t *scoreMapTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// makeScoreMapKey
////////////////////////////////////////////////////////////////////////

type MakeScoreMapKeyTest struct {
	scoreMapTest
}

func init() { RegisterTestSuite(&MakeScoreMapKeyTest{}) }

func (t *MakeScoreMapKeyTest) call() (key *state.ScoreMapKey) {
	var err error

	// Set up the Info field.
	t.node.Info, err = os.Lstat(path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)

	// Call through.
	key = makeScoreMapKey(&t.node, &t.clock)

	return
}

func (t *MakeScoreMapKeyTest) Directory() {
	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Symlink() {
	var err error

	// Set up
	t.node.RelPath = "foo"

	err = os.Symlink("blah", path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)

	// Call
	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) RecentlyModified() {
	var err error
	var key *state.ScoreMapKey

	// Set up
	t.node.RelPath = "foo"

	f, err := os.Create(path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)
	defer f.Close()

	fi, err := f.Stat()
	AssertEq(nil, err)

	// A short while ago
	t.clock.SetTime(fi.ModTime().Add(10 * time.Second))

	key = t.call()
	ExpectEq(nil, key)

	// Now
	t.clock.SetTime(fi.ModTime())

	key = t.call()
	ExpectEq(nil, key)

	// A short while in the future
	t.clock.SetTime(fi.ModTime().Add(-10 * time.Second))

	key = t.call()
	ExpectEq(nil, key)

	// Far in the future
	t.clock.SetTime(fi.ModTime().Add(-365 * 24 * time.Hour))

	key = t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Valid() {
	var err error

	// Set up
	t.node.RelPath = "foo"

	f, err := os.Create(path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)
	defer f.Close()

	_, err = f.Write([]byte("tacoburrito"))
	AssertEq(nil, err)

	fi, err := f.Stat()
	AssertEq(nil, err)

	// Call
	key := t.call()
	AssertNe(nil, key)

	ExpectEq(t.node.RelPath, key.Path)
	ExpectEq(fi.Mode()&os.ModePerm, key.Permissions)
	ExpectEq(fi.Sys().(*syscall.Stat_t).Uid, key.Uid)
	ExpectEq(fi.Sys().(*syscall.Stat_t).Gid, key.Gid)
	ExpectThat(key.MTime, timeutil.TimeEq(fi.ModTime()))
	ExpectEq(fi.Sys().(*syscall.Stat_t).Ino, key.Inode)
	ExpectEq(fi.Size(), key.Size)
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
	var err error

	// Make sure the node is eligible by default.
	t.node.RelPath = "foo"

	f, err := os.Create(path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)
	defer f.Close()

	t.node.Info, err = f.Stat()
	AssertEq(nil, err)

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
	t.node.Info, err = os.Stat(t.dir)
	AssertEq(nil, err)

	// Call. Nothing should be changed about the Scores field.
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Scores)
	ExpectEq(nil, t.node.scoreMapKey)
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

	ExpectThat(t.node.Scores, ElementsAre(score1, score2))
	ExpectEq(nil, t.node.scoreMapKey)
}

func (t *ConsultScoreMapTest) AbsentInScoreMap() {
	var err error

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectEq(nil, t.node.Scores)
	ExpectThat(t.node.scoreMapKey, Pointee(DeepEquals(*t.expectedKey)))
}

////////////////////////////////////////////////////////////////////////
// updateScoreMap
////////////////////////////////////////////////////////////////////////

type UpdateScoreMapTest struct {
	scoreMapTest
	expectedKey *state.ScoreMapKey
}

func init() { RegisterTestSuite(&UpdateScoreMapTest{}) }

func (t *UpdateScoreMapTest) SetUp(ti *TestInfo) {
	t.scoreMapTest.SetUp(ti)
	var err error

	// Make sure the node is eligible by default.
	t.node.RelPath = "foo"

	f, err := os.Create(path.Join(t.dir, t.node.RelPath))
	AssertEq(nil, err)
	defer f.Close()

	t.node.Info, err = f.Stat()
	AssertEq(nil, err)

	t.expectedKey = makeScoreMapKey(&t.node, &t.clock)
	AssertNe(nil, t.expectedKey)
}

func (t *UpdateScoreMapTest) call() (err error) {
	nodesIn := make(chan *fsNode, 1)
	nodesIn <- &t.node
	close(nodesIn)

	err = updateScoreMap(t.ctx, t.scoreMap, nodesIn)
	return
}

func (t *UpdateScoreMapTest) NodeNotEligible() {
	var err error

	// Make the node appear as a directory.
	t.node.Info, err = os.Stat(t.dir)
	AssertEq(nil, err)

	// Nothing bad should happen.
	err = t.call()
	AssertEq(nil, err)
}

func (t *UpdateScoreMapTest) NodeWasAlreadyPresent() {
	var err error

	// Prepare
	t.node.Scores = []blob.Score{
		blob.ComputeScore([]byte("taco")),
		blob.ComputeScore([]byte("burrito")),
	}

	t.node.scoreMapKey = nil

	// Call
	err = t.call()
	AssertEq(nil, err)
	ExpectEq(nil, t.scoreMap.Get(*t.expectedKey))
}

func (t *UpdateScoreMapTest) NodeWasntAlreadyPresent() {
	var err error

	// Prepare
	t.node.Scores = []blob.Score{
		blob.ComputeScore([]byte("taco")),
		blob.ComputeScore([]byte("burrito")),
	}

	t.node.scoreMapKey = t.expectedKey

	// Call
	err = t.call()
	AssertEq(nil, err)

	ExpectThat(
		t.scoreMap.Get(*t.expectedKey),
		ElementsAre(t.node.Scores[0], t.node.Scores[1]))
}

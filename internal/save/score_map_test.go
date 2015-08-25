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
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/state"
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
	key = makeScoreMapKey(&t.node, &t.clock)
	return
}

func (t *MakeScoreMapKeyTest) Directory() {
	var err error

	// Set up
	t.node.Info, err = os.Lstat(t.dir)
	AssertEq(nil, err)

	// Call
	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Symlink() {
	var err error

	// Set up
	p := path.Join(t.dir, "foo")
	err = os.Symlink("blah", p)
	AssertEq(nil, err)

	t.node.Info, err = os.Lstat(p)
	AssertEq(nil, err)

	// Call
	key := t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) RecentlyModified() {
	var err error
	var key *state.ScoreMapKey

	// Set up
	p := path.Join(t.dir, "foo")

	f, err := os.Create(p)
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

	// In the future
	t.clock.SetTime(fi.ModTime().Add(-10 * time.Second))

	key = t.call()
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) Valid() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// consultScoreMap
////////////////////////////////////////////////////////////////////////

type ConsultScoreMapTest struct {
	scoreMapTest
}

func init() { RegisterTestSuite(&ConsultScoreMapTest{}) }

func (t *ConsultScoreMapTest) call() (err error) {
	err = errors.New("TODO")
	return
}

func (t *ConsultScoreMapTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

////////////////////////////////////////////////////////////////////////
// updateScoreMap
////////////////////////////////////////////////////////////////////////

type UpdateScoreMapTest struct {
	scoreMapTest
}

func init() { RegisterTestSuite(&UpdateScoreMapTest{}) }

func (t *UpdateScoreMapTest) call() (err error) {
	err = errors.New("TODO")
	return
}

func (t *UpdateScoreMapTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

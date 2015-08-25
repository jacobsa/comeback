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
	clock    timeutil.Clock

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
	t.clock = timeutil.RealClock()

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

func (t *MakeScoreMapKeyTest) Directory() {
	var err error

	// Set up
	t.node.Info, err = os.Lstat(t.dir)
	AssertEq(nil, err)

	// Call
	key := makeScoreMapKey(&t.node)
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
	key := makeScoreMapKey(&t.node)
	ExpectEq(nil, key)
}

func (t *MakeScoreMapKeyTest) RecentlyModified() {
	AssertTrue(false, "TODO")
}

func (t *MakeScoreMapKeyTest) ModifiedInFuture() {
	AssertTrue(false, "TODO")
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

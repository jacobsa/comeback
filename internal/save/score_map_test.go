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

	"github.com/jacobsa/comeback/internal/fs"
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

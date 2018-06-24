// Copyright 2012 Aaron Jacobs. All Rights Reserved.
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

package state_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/comeback/internal/util"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestState(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type StateTest struct {
	s state.State
}

func init() { RegisterTestSuite(&StateTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StateTest) RoundTrip() {
	t.s.ExistingScores = util.NewStringSet()
	t.s.ExistingScores.Add("taco")
	t.s.ExistingScores.Add("burrito")
	t.s.RelistTime = time.Now().Round(0)

	t.s.ScoresForFiles = state.NewScoreMap()
	key := state.ScoreMapKey{Path: "queso"}
	scores := []blob.Score{blob.ComputeScore([]byte("foo"))}
	t.s.ScoresForFiles.Set(key, scores)

	// Save
	buf := new(bytes.Buffer)
	AssertEq(nil, state.SaveState(buf, t.s))

	// Load
	loaded, err := state.LoadState(buf)
	AssertEq(nil, err)

	ExpectTrue(loaded.ExistingScores.Contains("taco"))
	ExpectTrue(loaded.ExistingScores.Contains("burrito"))
	ExpectFalse(loaded.ExistingScores.Contains("enchilada"))
	ExpectThat(loaded.RelistTime, timeutil.TimeEq(t.s.RelistTime))

	ExpectThat(loaded.ScoresForFiles.Get(key), DeepEquals(scores))
}

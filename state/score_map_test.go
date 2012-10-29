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
	"github.com/jacobsa/comeback/state"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestScoreMap(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ScoreMapTest struct {
	m state.ScoreMap
}

func init() { RegisterTestSuite(&ScoreMapTest{}) }

func (t *ScoreMapTest) SetUp(i *TestInfo) {
	t.m = state.NewScoreMap()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ScoreMapTest) EmptyMap() {
	var key state.ScoreMapKey

	key = state.ScoreMapKey{Path: ""}
	ExpectEq(nil, t.m.Get(key))

	key = state.ScoreMapKey{Path: "taco"}
	ExpectEq(nil, t.m.Get(key))

	key = state.ScoreMapKey{Path: "burrito"}
	ExpectEq(nil, t.m.Get(key))
}

func (t *ScoreMapTest) SomeElements() {
	// Add taco
	tacoKey := state.ScoreMapKey{
		Path: "taco",
		Size: 17,
	}

	tacoScores := []blob.Score{
		blob.ComputeScore("foo"),
		blob.ComputeScore("bar"),
	}

	t.m.Add(tacoKey, tacoScores)

	// Add burrito
	burritoKey := tacoKey
	burritoKey.Path = "burrito"

	burritoScores := []blob.Score{
		blob.ComputeScore("baz"),
	}

	t.m.Add(burritoKey, burritoScores)

	// Look up
	ExpectThat(t.m.Get(tacoKey), DeepEquals(tacoScores))
	ExpectThat(t.m.Get(burritoKey), DeepEquals(burritoScores))
	ExpectEq(nil, t.m.Get(state.ScoreMapKey{}))
}

func (t *ScoreMapTest) AddTwice() {
	ExpectEq("TODO", "")
}

func (t *ScoreMapTest) GobRoundTrip() {
	ExpectEq("TODO", "")
}

func (t *ScoreMapTest) DecodingOverwritesContents() {
	ExpectEq("TODO", "")
}

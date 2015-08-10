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
	"encoding/gob"
	"testing"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/state"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
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
	// Set taco
	tacoKey := state.ScoreMapKey{
		Path: "taco",
		Size: 17,
	}

	tacoScores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	t.m.Set(tacoKey, tacoScores)

	// Set burrito
	burritoKey := tacoKey
	burritoKey.Path = "burrito"

	burritoScores := []blob.Score{
		blob.ComputeScore([]byte("baz")),
	}

	t.m.Set(burritoKey, burritoScores)

	// Look up
	ExpectThat(t.m.Get(tacoKey), DeepEquals(tacoScores))
	ExpectThat(t.m.Get(burritoKey), DeepEquals(burritoScores))
	ExpectEq(nil, t.m.Get(state.ScoreMapKey{}))
}

func (t *ScoreMapTest) AddTwice() {
	key := state.ScoreMapKey{Path: "taco"}

	// First
	scores0 := []blob.Score{
		blob.ComputeScore([]byte("foo")),
	}

	t.m.Set(key, scores0)

	// Second
	scores1 := []blob.Score{
		blob.ComputeScore([]byte("bar")),
	}

	t.m.Set(key, scores1)

	// Look up
	ExpectThat(t.m.Get(key), DeepEquals(scores1))
}

func (t *ScoreMapTest) GobRoundTrip() {
	// Contents
	key0 := state.ScoreMapKey{Path: "taco"}
	scores0 := []blob.Score{blob.ComputeScore([]byte("foo"))}
	t.m.Set(key0, scores0)

	key1 := state.ScoreMapKey{Path: "burrito"}
	scores1 := []blob.Score{blob.ComputeScore([]byte("bar"))}
	t.m.Set(key1, scores1)

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.m))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decoded state.ScoreMap
	AssertEq(nil, decoder.Decode(&decoded))

	ExpectThat(decoded.Get(key0), DeepEquals(scores0))
	ExpectThat(decoded.Get(key1), DeepEquals(scores1))
	ExpectEq(nil, decoded.Get(state.ScoreMapKey{}))
}

func (t *ScoreMapTest) DecodingOverwritesContents() {
	key0 := state.ScoreMapKey{Path: "taco"}
	key1 := state.ScoreMapKey{Path: "burrito"}
	key2 := state.ScoreMapKey{Path: "enchilada"}

	// Source contents
	scores0 := []blob.Score{blob.ComputeScore([]byte("foo"))}
	scores1 := []blob.Score{blob.ComputeScore([]byte("bar"))}

	t.m.Set(key0, scores0)
	t.m.Set(key1, scores1)

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.m))

	// Destination
	decoded := state.NewScoreMap()
	decoded.Set(key0, scores1)
	decoded.Set(key2, scores0)

	// Decode
	decoder := gob.NewDecoder(buf)
	AssertEq(nil, decoder.Decode(&decoded))

	ExpectThat(decoded.Get(key0), DeepEquals(scores0))
	ExpectThat(decoded.Get(key1), DeepEquals(scores1))
	ExpectEq(nil, decoded.Get(key2))
}

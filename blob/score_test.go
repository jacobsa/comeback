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

package blob

import (
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestRegister(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func fromHex(h string) []byte {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(fmt.Sprintf("Invalid hex string: %s", h))
	}

	return b
}

func toInterfaces(buf []byte) []interface{} {
	result := []interface{}{}
	for _, b := range buf {
		result = append(result, b)
	}

	return result
}

type ScoreTest struct {}
func init() { RegisterTestSuite(&ScoreTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ScoreTest) EmptySlice() {
	data := []byte{}
	golden := "da39a3ee5e6b4b0d3255bfef95601890afd80709"

	score := ComputeScore(data)
	AssertNe(nil, score)

	hash := score.Sha1Hash()
	AssertNe(nil, hash)
	AssertEq(20, len(hash))
	ExpectThat(hash, ElementsAre(toInterfaces(fromHex(golden))...))

	AssertEq(golden, HexScore(score))
}

func (t *ScoreTest) HashStartsWithZero() {
	ExpectEq("TODO", "")
}

func (t *ScoreTest) HexHashStartsWithNonZeroNumber() {
	ExpectEq("TODO", "")
}

func (t *ScoreTest) HexHashStartsWithLetter() {
	ExpectEq("TODO", "")
}

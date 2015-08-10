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
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestRegister(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func fromHex(h string) (s Score) {
	b, err := hex.DecodeString(h)
	if err != nil {
		panic(fmt.Sprintf("Invalid hex string: %s", h))
	}

	AssertEq(ScoreLength, len(b))
	copy(s[:], b)

	return
}

type ScoreTest struct{}

func init() { RegisterTestSuite(&ScoreTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ScoreTest) EmptySlice() {
	data := []byte{}
	golden := "da39a3ee5e6b4b0d3255bfef95601890afd80709"

	score := ComputeScore(data)
	AssertEq(20, len(score))

	ExpectEq(golden, score.Hex())
	ExpectThat(score, DeepEquals(fromHex(golden)))

	parsed, err := ParseHexScore(golden)
	AssertEq(nil, err)
	ExpectThat(parsed, DeepEquals(score))
}

func (t *ScoreTest) HashStartsWithZero() {
	data := []byte("hello_5")
	golden := "086766b9ba6a30e3792c05b00c5fb0e85a18a040"

	score := ComputeScore(data)
	AssertEq(20, len(score))

	ExpectEq(golden, score.Hex())
	ExpectThat(score, DeepEquals(fromHex(golden)))

	parsed, err := ParseHexScore(golden)
	AssertEq(nil, err)
	ExpectThat(parsed, DeepEquals(score))
}

func (t *ScoreTest) HexHashStartsWithNonZeroNumber() {
	data := []byte("hello_0")
	golden := "3966a6c98206d4cda8fd000656ed4f279a35726b"

	score := ComputeScore(data)
	AssertEq(20, len(score))

	ExpectEq(golden, score.Hex())
	ExpectThat(score, DeepEquals(fromHex(golden)))

	parsed, err := ParseHexScore(golden)
	AssertEq(nil, err)
	ExpectThat(parsed, DeepEquals(score))
}

func (t *ScoreTest) HexHashStartsWithLetter() {
	data := []byte("foo_barbazqux")
	golden := "ccf73cc0bfe964b652934764f847699e4005205e"

	score := ComputeScore(data)
	AssertEq(20, len(score))

	ExpectEq(golden, score.Hex())
	ExpectThat(score, DeepEquals(fromHex(golden)))

	parsed, err := ParseHexScore(golden)
	AssertEq(nil, err)
	ExpectThat(parsed, DeepEquals(score))
}

func (t *ScoreTest) DataContainsNonUtf8() {
	data := []byte{0x4a, 0x80, 0x81, 0x82, 0x4b}
	golden := "2feba26855d9f4e8b76d36c34dc385c8afe622c8"

	score := ComputeScore(data)
	AssertEq(20, len(score))

	ExpectEq(golden, score.Hex())
	ExpectThat(score, DeepEquals(fromHex(golden)))

	parsed, err := ParseHexScore(golden)
	AssertEq(nil, err)
	ExpectThat(parsed, DeepEquals(score))
}

func (t *ScoreTest) ParseError_Short() {
	in := strings.Repeat("0", 39)
	_, err := ParseHexScore(in)

	ExpectThat(err, Error(HasSubstr(in)))
	ExpectThat(err, Error(HasSubstr("legal hex score")))
}

func (t *ScoreTest) ParseError_Long() {
	in := strings.Repeat("0", 41)
	_, err := ParseHexScore(in)

	ExpectThat(err, Error(HasSubstr(in)))
	ExpectThat(err, Error(HasSubstr("legal hex score")))
}

func (t *ScoreTest) ParseError_CapitalLetter() {
	in := strings.Repeat("0", 39) + "A"
	_, err := ParseHexScore(in)

	ExpectThat(err, Error(HasSubstr(in)))
	ExpectThat(err, Error(HasSubstr("legal hex score")))
}

func (t *ScoreTest) ParseError_NonHexCharacter() {
	in := strings.Repeat("0", 39) + "g"
	_, err := ParseHexScore(in)

	ExpectThat(err, Error(HasSubstr(in)))
	ExpectThat(err, Error(HasSubstr("legal hex score")))
}

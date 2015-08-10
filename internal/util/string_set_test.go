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

package util_test

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/jacobsa/comeback/internal/util"
	. "github.com/jacobsa/ogletest"
)

func TestStringSet(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type StringSetTest struct {
	set util.StringSet
}

func init() { RegisterTestSuite(&StringSetTest{}) }

func (t *StringSetTest) SetUp(i *TestInfo) {
	t.set = util.NewStringSet()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *StringSetTest) EmptySet() {
	ExpectFalse(t.set.Contains(""))
	ExpectFalse(t.set.Contains("taco"))
	ExpectFalse(t.set.Contains("burrito"))
}

func (t *StringSetTest) SomeElements() {
	t.set.Add("taco")
	t.set.Add("burrito")

	ExpectFalse(t.set.Contains(""))
	ExpectTrue(t.set.Contains("taco"))
	ExpectTrue(t.set.Contains("burrito"))
	ExpectFalse(t.set.Contains("enchilada"))
}

func (t *StringSetTest) AddTwice() {
	t.set.Add("taco")
	t.set.Add("taco")

	ExpectFalse(t.set.Contains(""))
	ExpectTrue(t.set.Contains("taco"))
	ExpectFalse(t.set.Contains("burrito"))
}

func (t *StringSetTest) GobRoundTrip() {
	// Contents
	t.set.Add("taco")
	t.set.Add("burrito")

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.set))

	// Decode
	decoder := gob.NewDecoder(buf)
	var decodedSet util.StringSet
	AssertEq(nil, decoder.Decode(&decodedSet))

	ExpectFalse(decodedSet.Contains(""))
	ExpectTrue(decodedSet.Contains("taco"))
	ExpectTrue(decodedSet.Contains("burrito"))
	ExpectFalse(decodedSet.Contains("enchilada"))
}

func (t *StringSetTest) DecodingOverwritesContents() {
	// Source contents
	t.set.Add("taco")

	// Encode
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	AssertEq(nil, encoder.Encode(&t.set))

	// Destination
	decodedSet := util.NewStringSet()
	decodedSet.Add("burrito")

	// Decode
	decoder := gob.NewDecoder(buf)
	AssertEq(nil, decoder.Decode(&decodedSet))

	ExpectTrue(decodedSet.Contains("taco"))
	ExpectFalse(decodedSet.Contains("burrito"))
}

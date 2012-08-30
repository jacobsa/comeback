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

package repr_test

import (
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestReprTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeLegalEntry() *fs.DirectoryEntry {
	entry := new(fs.DirectoryEntry)
	entry.Type = fs.TypeDirectory
	return entry
}

type MarshalTest struct {
}

func init() { RegisterTestSuite(&MarshalTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MarshalTest) NoEntries() {
	// Input
	in := []*fs.DirectoryEntry{}

	// Marshal
	d, err := repr.Marshal(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.Unmarshal(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	ExpectThat(out, ElementsAre())
}

func (t *MarshalTest) UnknownType() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[1].Type = 17

	// Marshal
	_, err := repr.Marshal(in)

	ExpectThat(err, Error(HasSubstr("EntryType")))
	ExpectThat(err, Error(HasSubstr("17")))
}

func (t *MarshalTest) PreservesTypes() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Type = fs.TypeFile
	in[1].Type = fs.TypeDirectory
	in[2].Type = fs.TypeSymlink

	// Marshal
	d, err := repr.Marshal(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.Unmarshal(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any(), Any()))

	ExpectEq(in[0].Type, out[0].Type)
	ExpectEq(in[1].Type, out[1].Type)
	ExpectEq(in[2].Type, out[2].Type)
}

func (t *MarshalTest) PreservesNames() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Name = "taco"
	in[1].Name = "burrito"

	// Marshal
	d, err := repr.Marshal(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.Unmarshal(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Name, out[0].Name)
	ExpectEq(in[1].Name, out[1].Name)
}

func (t *MarshalTest) PreservesPermissions() {
	ExpectEq("TODO", "")
}

func (t *MarshalTest) UnrepresentableModTime() {
	ExpectEq("TODO", "")
}

func (t *MarshalTest) PreservesModTimes() {
	ExpectEq("TODO", "")
}

func (t *MarshalTest) CopesWithLocationsInModTimes() {
	ExpectEq("TODO", "")
}

func (t *MarshalTest) PreservesScores() {
	ExpectEq("TODO", "")
}

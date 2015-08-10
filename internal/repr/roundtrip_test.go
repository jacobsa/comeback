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
	"os"
	"testing"
	"time"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestRoundtripTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeLegalEntry() *fs.DirectoryEntry {
	entry := new(fs.DirectoryEntry)
	entry.Type = fs.TypeDirectory
	return entry
}

type RoundtripTest struct {
}

func init() { RegisterTestSuite(&RoundtripTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *RoundtripTest) NoEntries() {
	// Input
	in := []*fs.DirectoryEntry{}

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	ExpectThat(out, ElementsAre())
}

func (t *RoundtripTest) UnknownType() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[1].Type = 17

	// Marshal
	_, err := repr.MarshalDir(in)

	ExpectThat(err, Error(HasSubstr("EntryType")))
	ExpectThat(err, Error(HasSubstr("17")))
}

func (t *RoundtripTest) PreservesInode() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Inode = 17
	in[1].Inode = 19

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertEq(2, len(out))

	ExpectEq(in[0].Inode, out[0].Inode)
	ExpectEq(in[1].Inode, out[1].Inode)
}

func (t *RoundtripTest) PreservesSize() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Size = 17
	in[1].Size = 19

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertEq(2, len(out))

	ExpectEq(in[0].Size, out[0].Size)
	ExpectEq(in[1].Size, out[1].Size)
}

func (t *RoundtripTest) PreservesTypes() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Type = fs.TypeFile
	in[1].Type = fs.TypeDirectory
	in[2].Type = fs.TypeSymlink
	in[3].Type = fs.TypeBlockDevice
	in[4].Type = fs.TypeCharDevice
	in[5].Type = fs.TypeNamedPipe

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertEq(6, len(out))

	ExpectEq(in[0].Type, out[0].Type)
	ExpectEq(in[1].Type, out[1].Type)
	ExpectEq(in[2].Type, out[2].Type)
	ExpectEq(in[3].Type, out[3].Type)
	ExpectEq(in[4].Type, out[4].Type)
	ExpectEq(in[5].Type, out[5].Type)
}

func (t *RoundtripTest) PreservesNames() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Name = "taco"
	in[1].Name = "burrito"

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Name, out[0].Name)
	ExpectEq(in[1].Name, out[1].Name)
}

func (t *RoundtripTest) PreservesPermissions() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Permissions = 0644 | os.ModeSticky | os.ModeSetuid
	in[1].Permissions = 0755 | os.ModeSetgid

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Permissions, out[0].Permissions)
	ExpectEq(in[1].Permissions, out[1].Permissions)
}

func (t *RoundtripTest) PreservesUids() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Uid = 17
	in[1].Uid = 19

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Uid, out[0].Uid)
	ExpectEq(in[1].Uid, out[1].Uid)
}

func (t *RoundtripTest) PreservesUsernames() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	s := "taco"
	in[0].Username = &s

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	AssertNe(nil, out[0].Username)
	ExpectEq(*in[0].Username, *out[0].Username)
	ExpectEq(nil, out[1].Username)
}

func (t *RoundtripTest) PreservesGids() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Gid = 17
	in[1].Gid = 19

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Gid, out[0].Gid)
	ExpectEq(in[1].Gid, out[1].Gid)
}

func (t *RoundtripTest) PreservesGroupnames() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	s := "taco"
	in[0].Groupname = &s

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	AssertNe(nil, out[0].Groupname)
	ExpectEq(*in[0].Groupname, *out[0].Groupname)
	ExpectEq(nil, out[1].Groupname)
}

func (t *RoundtripTest) PreservesModTimes() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].MTime = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	in[1].MTime = time.Date(1985, time.March, 18, 15, 33, 0, 0, time.UTC)

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectTrue(in[0].MTime.Equal(out[0].MTime), "%v", out[0].MTime)
	ExpectTrue(in[1].MTime.Equal(out[1].MTime), "%v", out[1].MTime)
}

func (t *RoundtripTest) PreservesScores() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	score00 := blob.ComputeScore([]byte("taco"))
	score01 := blob.ComputeScore([]byte("burrito"))
	score10 := blob.ComputeScore([]byte("enchilada"))

	in[0].Scores = []blob.Score{score00, score01}
	in[1].Scores = []blob.Score{score10}

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	AssertThat(out[0].Scores, ElementsAre(Any(), Any()))
	ExpectEq(score00, out[0].Scores[0])
	ExpectEq(score01, out[0].Scores[1])

	AssertThat(out[1].Scores, ElementsAre(Any()))
	ExpectEq(score10, out[1].Scores[0])
}

func (t *RoundtripTest) PreservesHardLinkTargets() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	s := "taco"
	in[0].HardLinkTarget = &s

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	AssertNe(nil, out[0].HardLinkTarget)
	ExpectEq(*in[0].HardLinkTarget, *out[0].HardLinkTarget)
	ExpectEq(nil, out[1].HardLinkTarget)
}

func (t *RoundtripTest) PreservesSymlinkTargets() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Type = fs.TypeSymlink
	in[1].Type = fs.TypeSymlink

	in[0].Target = "taco"
	in[1].Target = "burrito"

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].Target, out[0].Target)
	ExpectEq(in[1].Target, out[1].Target)
}

func (t *RoundtripTest) PreservesDeviceNumbers() {
	// Input
	in := []*fs.DirectoryEntry{
		makeLegalEntry(),
		makeLegalEntry(),
	}

	in[0].Type = fs.TypeCharDevice
	in[1].Type = fs.TypeBlockDevice

	in[0].DeviceNumber = 17
	in[1].DeviceNumber = 19

	// Marshal
	d, err := repr.MarshalDir(in)
	AssertEq(nil, err)
	AssertNe(nil, d)

	// Unmarshal
	out, err := repr.UnmarshalDir(d)
	AssertEq(nil, err)
	AssertNe(nil, out)

	// Output
	AssertThat(out, ElementsAre(Any(), Any()))

	ExpectEq(in[0].DeviceNumber, out[0].DeviceNumber)
	ExpectEq(in[1].DeviceNumber, out[1].DeviceNumber)
}

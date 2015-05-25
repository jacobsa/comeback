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
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/repr/proto"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestMarshalTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type MarshalTest struct {
}

func init() { RegisterTestSuite(&MarshalTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MarshalTest) LeavesOutTargetForNonSymlinks() {
	// Input
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{Type: fs.TypeFile},
		&fs.DirectoryEntry{Type: fs.TypeDirectory},
	}

	// Call
	data, err := repr.MarshalDir(entries)
	AssertEq(nil, err)

	listingProto := new(repr_proto.DirectoryListingProto)
	err = proto.Unmarshal(data, listingProto)
	AssertEq(nil, err)

	AssertThat(listingProto.Entry, ElementsAre(Any(), Any()))
	ExpectEq(nil, listingProto.Entry[0].Target)
	ExpectEq(nil, listingProto.Entry[1].Target)
}

func (t *MarshalTest) LeavesOutDeviceNumberForNonDevices() {
	// Input
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{Type: fs.TypeFile},
		&fs.DirectoryEntry{Type: fs.TypeDirectory},
	}

	// Call
	data, err := repr.MarshalDir(entries)
	AssertEq(nil, err)

	listingProto := new(repr_proto.DirectoryListingProto)
	err = proto.Unmarshal(data, listingProto)
	AssertEq(nil, err)

	AssertThat(listingProto.Entry, ElementsAre(Any(), Any()))
	ExpectEq(nil, listingProto.Entry[0].DeviceNumber)
	ExpectEq(nil, listingProto.Entry[1].DeviceNumber)
}

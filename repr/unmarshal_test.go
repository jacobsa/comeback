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

	"github.com/golang/protobuf/proto"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/repr"
	"github.com/jacobsa/comeback/repr/proto"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestUnmarshalTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func computeScoreSlice(b []byte) (score []byte) {
	s := blob.ComputeScore(b)
	return s[:]
}

func makeLegalEntryProto() *repr_proto.DirectoryEntryProto {
	return &repr_proto.DirectoryEntryProto{}
}

type UnmarshalTest struct {
}

func init() { RegisterTestSuite(&UnmarshalTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *UnmarshalTest) JunkWireData() {
	// Input
	d := []byte("asdf")

	// Call
	_, err := repr.UnmarshalDir(d)

	ExpectThat(err, Error(HasSubstr("Parsing data")))
}

func (t *UnmarshalTest) InvalidTypeValue() {
	// Input
	listingProto := &repr_proto.DirectoryListingProto{
		Entry: []*repr_proto.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	listingProto.Entry[1].Type = repr_proto.DirectoryEntryProto_Type(17).Enum()

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	_, err = repr.UnmarshalDir(data)

	ExpectThat(err, Error(HasSubstr("Unrecognized")))
	ExpectThat(err, Error(HasSubstr("DirectoryEntryProto_Type")))
	ExpectThat(err, Error(HasSubstr("17")))
}

func (t *UnmarshalTest) UnknownTypeValue() {
	// Input
	listingProto := &repr_proto.DirectoryListingProto{
		Entry: []*repr_proto.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	listingProto.Entry[1].Type = repr_proto.DirectoryEntryProto_TYPE_UNKNOWN.Enum()

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	_, err = repr.UnmarshalDir(data)

	ExpectThat(err, Error(HasSubstr("Unrecognized")))
	ExpectThat(err, Error(HasSubstr("DirectoryEntryProto_Type")))
	ExpectThat(err, Error(HasSubstr("TYPE_UNKNOWN")))
}

func (t *UnmarshalTest) HashIsTooShort() {
	// Input
	listingProto := &repr_proto.DirectoryListingProto{
		Entry: []*repr_proto.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	listingProto.Entry[1].Blob = []*repr_proto.BlobInfoProto{
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
	}

	blob := listingProto.Entry[1].Blob[1]
	blob.Hash = blob.Hash[0 : len(blob.Hash)-1]

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	_, err = repr.UnmarshalDir(data)

	ExpectThat(err, Error(HasSubstr("hash length")))
	ExpectThat(err, Error(HasSubstr("19")))
}

func (t *UnmarshalTest) HashIsTooLong() {
	// Input
	listingProto := &repr_proto.DirectoryListingProto{
		Entry: []*repr_proto.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	listingProto.Entry[1].Blob = []*repr_proto.BlobInfoProto{
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
		&repr_proto.BlobInfoProto{Hash: computeScoreSlice([]byte{})},
	}

	blob := listingProto.Entry[1].Blob[1]
	blob.Hash = append(blob.Hash, 0x01)

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	_, err = repr.UnmarshalDir(data)

	ExpectThat(err, Error(HasSubstr("hash length")))
	ExpectThat(err, Error(HasSubstr("21")))
}

// Catch changes to the layout of os.FileMode, which we have hard-coded by
// virtue of relying on its bit placement in stable storage.
func (t *UnmarshalTest) PermissionsRegressionTest() {
	AssertEq(0777, os.ModePerm)

	// Input
	listingProto := &repr_proto.DirectoryListingProto{
		Entry: []*repr_proto.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	setuid := 1 << 23
	setgid := 1 << 22
	sticky := 1 << 20

	listingProto.Entry[0].Permissions = proto.Uint32(uint32(0751 | setuid))
	listingProto.Entry[1].Permissions = proto.Uint32(uint32(0157 | setgid))
	listingProto.Entry[2].Permissions = proto.Uint32(uint32(0000 | sticky))

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	entries, err := repr.UnmarshalDir(data)
	AssertEq(nil, err)

	AssertThat(entries, ElementsAre(Any(), Any(), Any()))
	ExpectEq(0751|os.ModeSetuid, entries[0].Permissions)
	ExpectEq(0157|os.ModeSetgid, entries[1].Permissions)
	ExpectEq(0000|os.ModeSticky, entries[2].Permissions)
}

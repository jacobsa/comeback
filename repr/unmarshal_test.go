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
	"code.google.com/p/goprotobuf/proto"
	"github.com/jacobsa/comeback/repr"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestUnmarshalTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeLegalEntryProto() *repr.DirectoryEntryProto {
	return &repr.DirectoryEntryProto{}
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
	_, err := repr.Unmarshal(d)

	ExpectThat(err, Error(HasSubstr("Parsing data")))
}

func (t *UnmarshalTest) UnknownTypeValue() {
	// Input
	listingProto := &repr.DirectoryListingProto{
		Entry: []*repr.DirectoryEntryProto{
			makeLegalEntryProto(),
			makeLegalEntryProto(),
			makeLegalEntryProto(),
		},
	}

	listingProto.Entry[0].Type = repr.DirectoryEntryProto_Type(17).Enum()

	data, err := proto.Marshal(listingProto)
	AssertEq(nil, err)

	// Call
	_, err = repr.Unmarshal(data)

	ExpectThat(err, Error(HasSubstr("Unrecognized")))
	ExpectThat(err, Error(HasSubstr("DirectoryEntryProto_Type")))
	ExpectThat(err, Error(HasSubstr("17")))
}

func (t *UnmarshalTest) HashIsTooShort() {
	ExpectEq("TODO", "")
}

func (t *UnmarshalTest) HashIsTooLong() {
	ExpectEq("TODO", "")
}

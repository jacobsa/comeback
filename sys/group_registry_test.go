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

package sys_test

import (
	"github.com/jacobsa/comeback/sys"
	"github.com/jacobsa/comeback/sys/group"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"log"
	"strconv"
	"testing"
)

func TestGroupRegistry(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type GroupRegistryTest struct {
	registry sys.GroupRegistry
}

func init() { RegisterTestSuite(&GroupRegistryTest{}) }

func (t *GroupRegistryTest) SetUp(i *TestInfo) {
	var err error
	if t.registry, err = sys.NewGroupRegistry(); err != nil {
		log.Fatalf("Creating registry: %v", err)
	}
}

func (t *GroupRegistryTest) UnknownGroupname() {
	_, err := t.registry.FindByName("jksdlhfy9823h4bnkqjsahdjkahsd")

	notFoundErr, ok := err.(sys.NotFoundError)
	AssertTrue(ok, "%v", err)
	ExpectThat(notFoundErr, HasSubstr("jksdlhfy9823h4bnkqjsahdjkahsd"))
	ExpectThat(notFoundErr, HasSubstr("unknown"))
}

func (t *GroupRegistryTest) UnknownGroupId() {
	_, err := t.registry.FindById(17192325)

	notFoundErr, ok := err.(sys.NotFoundError)
	AssertTrue(ok, "%v", err)
	ExpectThat(notFoundErr, HasSubstr("171923"))
	ExpectThat(notFoundErr, HasSubstr("unknown"))
}

func (t *GroupRegistryTest) LookUpCurrentGroup() {
	// Ask the os package for the current group.
	osGroup, err := group.Current()
	AssertEq(nil, err)

	AssertNe("", osGroup.Groupname)
	AssertNe("", osGroup.Gid)

	osGid, err := strconv.Atoi(osGroup.Gid)
	AssertEq(nil, err)
	AssertNe(0, osGid)

	// Look it up in both ways.
	groupname, err := t.registry.FindById(sys.GroupId(osGid))
	AssertEq(nil, err)
	ExpectEq(osGroup.Groupname, groupname)

	gid, err := t.registry.FindByName(osGroup.Groupname)
	AssertEq(nil, err)
	ExpectEq(sys.GroupId(osGid), gid)
}

func (t *GroupRegistryTest) LookUpWheelGroup() {
	// Ask the os package for the wheel group.
	osGroup, err := group.Lookup("wheel")
	AssertEq(nil, err)

	AssertNe("", osGroup.Groupname)
	AssertNe("", osGroup.Gid)

	osGid, err := strconv.Atoi(osGroup.Gid)
	AssertEq(nil, err)

	// Look it up in both ways.
	groupname, err := t.registry.FindById(sys.GroupId(osGid))
	AssertEq(nil, err)
	ExpectEq("wheel", groupname)

	gid, err := t.registry.FindByName("wheel")
	AssertEq(nil, err)
	ExpectEq(sys.GroupId(osGid), gid)
}

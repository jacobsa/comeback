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
	. "github.com/jacobsa/ogletest"
	"log"
	"os/user"
	"strconv"
	"testing"
)

func TestUserRegistry(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type UserRegistryTest struct {
	registry sys.UserRegistry
}

func init() { RegisterTestSuite(&UserRegistryTest{}) }

func (t *UserRegistryTest) SetUp(i *TestInfo) {
	var err error
	if t.registry, err = sys.NewUserRegistry(); err != nil {
		log.Fatalf("Creating registry: %v", err)
	}
}

func (t *UserRegistryTest) UnknownUsername() {
	ExpectEq("TODO", "")
}

func (t *UserRegistryTest) UnknownUserId() {
	ExpectEq("TODO", "")
}

func (t *UserRegistryTest) LookUpCurrentUser() {
	// Ask the os package for the current user.
	osUser, err := user.Current()
	AssertEq(nil, err)

	AssertNe("", osUser.Username)
	AssertNe("", osUser.Uid)

	osUid, err := strconv.Atoi(osUser.Uid)
	AssertEq(nil, err)
	AssertNe(0, osUid)

	// Look it up in both ways.
	username, err := t.registry.FindById(sys.UserId(osUid))
	AssertEq(nil, err)
	ExpectEq(osUser.Username, username)

	uid, err := t.registry.FindByName(osUser.Username)
	AssertEq(nil, err)
	ExpectEq(sys.UserId(osUid), uid)
}

func (t *UserRegistryTest) LookUpRootUser() {
	// Ask the os package for the root user.
	osUser, err := user.Lookup("root")
	AssertEq(nil, err)

	AssertNe("", osUser.Username)
	AssertNe("", osUser.Uid)

	osUid, err := strconv.Atoi(osUser.Uid)
	AssertEq(nil, err)
	AssertNe(0, osUid)

	// Look it up in both ways.
	username, err := t.registry.FindById(sys.UserId(osUid))
	AssertEq(nil, err)
	ExpectEq("root", username)

	uid, err := t.registry.FindByName("root")
	AssertEq(nil, err)
	ExpectEq(sys.UserId(osUid), uid)
}

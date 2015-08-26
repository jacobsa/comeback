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
	"os/user"
	"strconv"
	"testing"

	"github.com/jacobsa/comeback/internal/sys"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
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
	t.registry = sys.NewUserRegistry()
}

func (t *UserRegistryTest) UnknownUsername() {
	_, err := t.registry.FindByName("jksdlhfy9823h4bnkqjsahdjkahsd")

	notFoundErr, ok := err.(sys.NotFoundError)
	AssertTrue(ok, "%v", err)
	ExpectThat(notFoundErr, HasSubstr("jksdlhfy9823h4bnkqjsahdjkahsd"))
	ExpectThat(notFoundErr, HasSubstr("unknown"))
}

func (t *UserRegistryTest) UnknownUserId() {
	_, err := t.registry.FindById(17192325)

	notFoundErr, ok := err.(sys.NotFoundError)
	AssertTrue(ok, "%v", err)
	ExpectThat(notFoundErr, HasSubstr("171923"))
	ExpectThat(notFoundErr, HasSubstr("unknown"))
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

	// Look it up in both ways.
	username, err := t.registry.FindById(sys.UserId(osUid))
	AssertEq(nil, err)
	ExpectEq("root", username)

	uid, err := t.registry.FindByName("root")
	AssertEq(nil, err)
	ExpectEq(sys.UserId(osUid), uid)
}

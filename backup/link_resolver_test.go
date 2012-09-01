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

package backup_test

import (
	"github.com/jacobsa/comeback/backup"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestLinkResolver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type LinkResolverTest struct {
	resolver backup.LinkResolver
}

func init() { RegisterTestSuite(&LinkResolverTest{}) }

func (t *LinkResolverTest) SetUp(i *TestInfo) {
	t.resolver = backup.NewLinkResolver()
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *LinkResolverTest) DifferingDevicesSameInode() {
	AssertEq(nil, t.resolver.Register(17, 19, "taco"))
	ExpectEq(nil, t.resolver.Register(23, 19, "burrito"))
}

func (t *LinkResolverTest) DifferingInodesSameDevice() {
	AssertEq(nil, t.resolver.Register(17, 19, "taco"))
	ExpectEq(nil, t.resolver.Register(17, 23, "burrito"))
}

func (t *LinkResolverTest) SameBoth() {
	AssertEq(nil, t.resolver.Register(17, 19, "taco"))
	ExpectThat(t.resolver.Register(17, 19, "burrito"), Pointee(Equals("taco")))
	ExpectThat(t.resolver.Register(17, 19, "enchilada"), Pointee(Equals("taco")))
	ExpectThat(t.resolver.Register(17, 19, "queso"), Pointee(Equals("taco")))
}

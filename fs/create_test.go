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

package fs_test

import (
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestCreate(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// CreateNamedPipe
////////////////////////////////////////////////////////////////////////

type CreateNamedPipeTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&CreateNamedPipeTest{}) }

func (t *CreateNamedPipeTest) DoesFoo() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// CreateBlockDevice
////////////////////////////////////////////////////////////////////////

type CreateBlockDeviceTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&CreateBlockDeviceTest{}) }

func (t *CreateBlockDeviceTest) DoesFoo() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// CreateCharDevice
////////////////////////////////////////////////////////////////////////

type CreateCharDeviceTest struct {
	fileSystemTest
}

func init() { RegisterTestSuite(&CreateCharDeviceTest{}) }

func (t *CreateCharDeviceTest) DoesFoo() {
	ExpectEq("TODO", "")
}

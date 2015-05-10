// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package wiring_test

import (
	"testing"

	. "github.com/jacobsa/ogletest"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type IntegrationTest struct {
}

func init() { RegisterTestSuite(&IntegrationTest{}) }

func (t *IntegrationTest) SetUp(i *TestInfo) {
	panic("TODO")
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *IntegrationTest) WrongPasswordForRegistry() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) WrongPasswordForDirSaver() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) WrongPasswordForDirRestorer() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) EmptyDirectory() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) SingleFile() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) SingleEmptySubDir() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) DecentHierarchy() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) StableResult() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) Symlinks() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) HardLinks() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) Permissions() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) OwnershipInfo() {
	AssertFalse(true, "TODO")
}

func (t *IntegrationTest) Mtime() {
	AssertFalse(true, "TODO")
}

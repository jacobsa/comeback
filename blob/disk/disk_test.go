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

package disk_test

import (
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/disk"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestDisk(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type diskStoreTest struct {
	basePath string
	fs       mock_fs.MockFileSystem
	store    blob.Store
}

func (t *diskStoreTest) SetUp(i *TestInfo) {
	var err error

	t.basePath = "/foo/bar"
	t.fs = mock_fs.NewMockFileSystem(i.MockController, "fs")

	t.store, err = disk.NewDiskBlobStore(t.basePath, t.fs)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

type StoreTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&StoreTest{}) }

func (t *StoreTest) CallsFileSystem() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) FileSystemReturnsError() {
	ExpectEq("TODO", "")
}

func (t *StoreTest) FileSystemSaysOkay() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type LoadTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&LoadTest{}) }

func (t *LoadTest) CallsFileSystem() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) FileSystemReturnsError() {
	ExpectEq("TODO", "")
}

func (t *LoadTest) FileSystemSucceeds() {
	ExpectEq("TODO", "")
}

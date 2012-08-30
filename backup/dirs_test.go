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
	"github.com/jacobsa/comeback/backup/mock"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestRegisterDirsTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type DirectorySaverTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem
	fileSaver mock_backup.MockFileSaver
	wrapped mock_backup.MockDirectorySaver

	dirSaver backup.DirectorySaver

	score blob.Score
	err error
}

func init() { RegisterTestSuite(&DirectorySaverTest{}) }

func (t *DirectorySaverTest) SetUp(i *TestInfo) {
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.fileSaver = mock_backup.NewMockFileSaver(i.MockController, "fileSaver")
	t.wrapped = mock_backup.NewMockDirectorySaver(i.MockController, "wrapped")

	t.dirSaver, _ = backup.NewDirectorySaver(
		t.blobStore,
		t.fileSystem,
		t.fileSaver,
		t.wrapped,
	)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirectorySaverTest) DoesFoo() {
	ExpectEq("TODO", "")
}

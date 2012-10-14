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

func TestDirectoryRestorer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type DirectoryRestorerTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem
	fileRestorer mock_backup.MockFileRestorer
	wrapped mock_backup.MockDirectoryRestorer

	dirRestorer backup.DirectoryRestorer

	score blob.Score
	basePath string
	relPath string

	err    error
}

func init() { RegisterTestSuite(&DirectoryRestorerTest{}) }

func (t *DirectoryRestorerTest) SetUp(i *TestInfo) {
	var err error

	// Create dependencies.
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.fileRestorer = mock_backup.NewMockFileRestorer(i.MockController, "fileRestorer")
	t.wrapped = mock_backup.NewMockDirectoryRestorer(i.MockController, "wrapped")

	// Create restorer.
	t.dirRestorer, err = backup.NewNonRecursiveDirectoryRestorer(
		t.blobStore,
		t.fileSystem,
		t.fileRestorer,
		t.wrapped,
	)

	AssertEq(nil, err)
}

func (t *DirectoryRestorerTest) call() {
	t.err = t.dirRestorer.RestoreDirectory(t.score, t.basePath, t.relPath)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirectoryRestorerTest) DoesFoo() {
	ExpectEq("TODO", "")
}

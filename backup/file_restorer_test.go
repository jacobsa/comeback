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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/ogletest"
	"os"
	"testing"
)

func TestFileRestorer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type FileRestorerTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem

	fileRestorer backup.FileRestorer

	scores []blob.Score
	path string
	perms os.FileMode

	err    error
}

func init() { RegisterTestSuite(&FileRestorerTest{}) }

func (t *FileRestorerTest) SetUp(i *TestInfo) {
	var err error

	// Create dependencies.
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")

	// Create restorer.
	t.fileRestorer, err = backup.NewFileRestorer(t.blobStore, t.fileSystem)
	AssertEq(nil, err)
}

func (t *FileRestorerTest) call() {
	t.err = t.fileRestorer.RestoreFile(t.scores, t.path, t.perms)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileRestorerTest) DoesFoo() {
	ExpectEq("TODO", "")
}

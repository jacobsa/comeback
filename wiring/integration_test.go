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

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/gcloud/gcs"
	. "github.com/jacobsa/ogletest"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Wiring
////////////////////////////////////////////////////////////////////////

type WiringTest struct {
}

func init() { RegisterTestSuite(&WiringTest{}) }

func (t *WiringTest) WrongPasswordForRegistry() {
	AssertFalse(true, "TODO")
}

func (t *WiringTest) WrongPasswordForDirSaver() {
	AssertFalse(true, "TODO")
}

func (t *WiringTest) WrongPasswordForDirRestorer() {
	AssertFalse(true, "TODO")
}

////////////////////////////////////////////////////////////////////////
// Saving and restoring
////////////////////////////////////////////////////////////////////////

type SaveAndRestoreTest struct {
	bucket gcs.Bucket

	// Temporary directories for saving from and restoring to.
	src string
	dst string
}

var _ SetUpInterface = &SaveAndRestoreTest{}
var _ TearDownInterface = &SaveAndRestoreTest{}

func init() { RegisterTestSuite(&SaveAndRestoreTest{}) }

func (t *SaveAndRestoreTest) SetUp(i *TestInfo) {
	panic("TODO")
}

func (t *SaveAndRestoreTest) TearDown() {
	panic("TODO")
}

// Make a backup of the contents of t.src into t.bucket, returning a score for
// the root listing.
func (t *SaveAndRestoreTest) save() (score blob.Score, err error) {
	panic("TODO")
}

// Restore a backup with the given root listing into t.dst.
func (t *SaveAndRestoreTest) restore(score blob.Score) (err error) {
	panic("TODO")
}

func (t *SaveAndRestoreTest) EmptyDirectory() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) SingleFile() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) SingleEmptySubDir() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) DecentHierarchy() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) StableResult() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) Symlinks() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) HardLinks() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) Permissions() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) OwnershipInfo() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) Mtime() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) BackupExclusions() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) MissingBlob() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) CorruptedBlob() {
	AssertFalse(true, "TODO")
}

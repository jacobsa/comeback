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
	"io/ioutil"
	"os"
	"testing"

	"github.com/googlecloudplatform/gcsfuse/timeutil"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/state"
	"github.com/jacobsa/comeback/util"
	"github.com/jacobsa/comeback/wiring"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestIntegration(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Common
////////////////////////////////////////////////////////////////////////

const password = "some password"

type commonTest struct {
	bucket gcs.Bucket
}

func (t *commonTest) SetUp(ti *TestInfo) {
	t.bucket = gcsfake.NewFakeBucket(timeutil.RealClock(), "")
}

////////////////////////////////////////////////////////////////////////
// Wiring
////////////////////////////////////////////////////////////////////////

type WiringTest struct {
	commonTest
}

func init() { RegisterTestSuite(&WiringTest{}) }

func (t *WiringTest) WrongPassword() {
	const wrongPassword = password + "bar"
	var err error

	// Create a registry in the bucket with the "official" password.
	_, _, err = wiring.MakeRegistryAndCrypter(password, t.bucket)
	AssertEq(nil, err)

	// Using a different password to do the same should fail.
	_, _, err = wiring.MakeRegistryAndCrypter(wrongPassword, t.bucket)
	ExpectThat(err, Error(HasSubstr("password is incorrect")))

	// Ditto with the dir saver.
	_, err = wiring.MakeDirSaver(
		wrongPassword,
		t.bucket,
		util.NewStringSet(),
		state.NewScoreMap())
	ExpectThat(err, Error(HasSubstr("password is incorrect")))

	// And the dir restorer.
	_, err = wiring.MakeDirRestorer(wrongPassword, t.bucket)
	ExpectThat(err, Error(HasSubstr("password is incorrect")))
}

////////////////////////////////////////////////////////////////////////
// Saving and restoring
////////////////////////////////////////////////////////////////////////

type SaveAndRestoreTest struct {
	commonTest

	// Temporary directories for saving from and restoring to.
	src string
	dst string
}

var _ SetUpInterface = &SaveAndRestoreTest{}
var _ TearDownInterface = &SaveAndRestoreTest{}

func init() { RegisterTestSuite(&SaveAndRestoreTest{}) }

func (t *SaveAndRestoreTest) SetUp(ti *TestInfo) {
	var err error
	t.commonTest.SetUp(ti)

	// Create the temporary directories.
	t.src, err = ioutil.TempDir("", "comeback_integration_test")
	AssertEq(nil, err)

	t.dst, err = ioutil.TempDir("", "comeback_integration_test")
	AssertEq(nil, err)
}

func (t *SaveAndRestoreTest) TearDown() {
	// Remove the temporary directories.
	ExpectEq(nil, os.RemoveAll(t.src))
	ExpectEq(nil, os.RemoveAll(t.dst))
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

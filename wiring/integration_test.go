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
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
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
// Helpers
////////////////////////////////////////////////////////////////////////

func randHex(n int) (s string) {
	if n%2 != 0 {
		panic(fmt.Sprintf("Invalid length: %v", n))
	}

	b := make([]byte, n/2)
	_, err := cryptorand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("cryptorand.Read: %v", err))
		return
	}

	s = hex.EncodeToString(b)
	return
}

func addRandomFile(dir string) (err error) {
	// Choose a random length that may or may not straddle multiple chunks.
	length := int(rand.Int31n(3 * fileChunkSize))

	// Generate contents.
	contents := make([]byte, length)
	_, err = cryptorand.Read(contents)
	if err != nil {
		err = fmt.Errorf("rand.Read: %v", err)
		return
	}

	// Write out a file.
	err = ioutil.WriteFile(path.Join(dir, randHex(16)), contents, 0400)
	if err != nil {
		err = fmt.Errorf("WriteFile: %v", err)
		return
	}

	return
}

// Put random files into a directory, recursing into two further children up to
// some limit.
func populateDir(dir string, depth int) (err error) {
	const depthLimit = 5

	// Add files.
	const numFiles = 10
	for i := 0; i < numFiles; i++ {
		err = addRandomFile(dir)
		if err != nil {
			err = fmt.Errorf("depth %v, addFile: %v", depth, err)
			return
		}
	}

	// Add sub-dirs if appropriate.
	if depth < depthLimit {
		const numSubdirs = 2
		for i := 0; i < numSubdirs; i++ {
			c := path.Join(dir, randHex(16))
			err = os.Mkdir(c, 0700)
			if err != nil {
				err = fmt.Errorf("Mkdir: %v", err)
				return
			}

			err = populateDir(c, depth+1)
			if err != nil {
				return
			}
		}
	}

	return
}

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
		1<<24,
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

const fileChunkSize = 1 << 12

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
	// Create the dir saver.
	dirSaver, err := wiring.MakeDirSaver(
		password,
		t.bucket,
		fileChunkSize,
		util.NewStringSet(),
		state.NewScoreMap())

	if err != nil {
		err = fmt.Errorf("MakeDirSaver: %v", err)
		return
	}

	// Save the source directory.
	score, err = dirSaver.Save(t.src, "", nil)
	if err != nil {
		err = fmt.Errorf("Save: %v", err)
		return
	}

	// Flush to stable storage.
	err = dirSaver.Flush()
	if err != nil {
		err = fmt.Errorf("Flush: %v", err)
		return
	}

	return
}

// Restore a backup with the given root listing into t.dst.
func (t *SaveAndRestoreTest) restore(score blob.Score) (err error) {
	// Create the dir restorer.
	dirRestorer, err := wiring.MakeDirRestorer(password, t.bucket)
	if err != nil {
		err = fmt.Errorf("MakeDirRestorer: %v", err)
		return
	}

	// Call it.
	err = dirRestorer.RestoreDirectory(score, t.dst, "")
	if err != nil {
		err = fmt.Errorf("RestoreDirectory: %v", err)
		return
	}

	return
}

func (t *SaveAndRestoreTest) EmptyDirectory() {
	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Check.
	entries, err := ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	ExpectEq(0, len(entries))
}

func (t *SaveAndRestoreTest) SingleSmallFile() {
	const contents = "taco"

	var fi os.FileInfo
	var err error

	// Create.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte(contents), 0400)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Read entries.
	entries, err := ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectFalse(fi.IsDir())

	// Read the file.
	b, err := ioutil.ReadFile(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))
}

func (t *SaveAndRestoreTest) SingleLargeFile() {
	var fi os.FileInfo
	var err error

	// Set up contents that span multiple chunks, including a partial one at the
	// end.
	contents := strings.Repeat("baz", 7*fileChunkSize+3)

	// Create.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte(contents), 0400)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Read entries.
	entries, err := ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectEq(len(contents), fi.Size())
	ExpectFalse(fi.IsDir())

	// Read the file.
	b, err := ioutil.ReadFile(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	if string(b) != contents {
		AddFailure("Contents mismatch")
	}
}

func (t *SaveAndRestoreTest) SingleEmptySubDir() {
	var entries []os.FileInfo
	var fi os.FileInfo
	var err error

	// Create.
	err = os.Mkdir(path.Join(t.src, "foo"), 0500)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Read entries (root).
	entries, err = ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Read entries (sub-dir).
	entries, err = ioutil.ReadDir(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(0, len(entries))
}

func (t *SaveAndRestoreTest) MultipleFilesAndDirs() {
	var b []byte
	var entries []os.FileInfo
	var fi os.FileInfo
	var err error

	// Create.
	AssertEq(nil, os.Mkdir(path.Join(t.src, "foo"), 0700))
	AssertEq(nil, os.Mkdir(path.Join(t.src, "bar"), 0700))
	AssertEq(nil, ioutil.WriteFile(path.Join(t.src, "baz"), []byte("aa"), 0400))
	AssertEq(nil, ioutil.WriteFile(path.Join(t.src, "bar/qux"), []byte("b"), 0400))

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Read entries (root).
	entries, err = ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	AssertEq(3, len(entries))

	fi = entries[0]
	ExpectEq("bar", fi.Name())
	ExpectTrue(fi.IsDir())

	fi = entries[1]
	ExpectEq("baz", fi.Name())
	ExpectFalse(fi.IsDir())
	ExpectEq(2, fi.Size())

	fi = entries[2]
	ExpectEq("foo", fi.Name())
	ExpectTrue(fi.IsDir())

	// Read entries (foo)
	entries, err = ioutil.ReadDir(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(0, len(entries))

	// Read entries (bar)
	entries, err = ioutil.ReadDir(path.Join(t.dst, "bar"))
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("qux", fi.Name())
	ExpectFalse(fi.IsDir())
	ExpectEq(1, fi.Size())

	// Read file (baz)
	b, err = ioutil.ReadFile(path.Join(t.dst, "baz"))
	AssertEq(nil, err)
	ExpectEq("aa", string(b))

	// Read file (qux)
	b, err = ioutil.ReadFile(path.Join(t.dst, "bar/qux"))
	AssertEq(nil, err)
	ExpectEq("b", string(b))
}

func (t *SaveAndRestoreTest) ResultScoreIsStable() {
	var err error

	// Ensure we get some parallelism for the duration of this test, in hopes of
	// exposing races.
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(16))

	// Set up random contents.
	err = populateDir(t.src, 0)
	AssertEq(nil, err)

	// Save multiple times.
	score0, err := t.save()
	AssertEq(nil, err)

	score1, err := t.save()
	AssertEq(nil, err)

	score2, err := t.save()
	AssertEq(nil, err)

	// The output should be the same each time.
	ExpectEq(score0, score1)
	ExpectEq(score0, score2)
}

func (t *SaveAndRestoreTest) HardLinks() {
	const contents = "taco"
	var err error

	// Create.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte(contents), 0400)
	AssertEq(nil, err)

	err = os.Link(path.Join(t.src, "foo"), path.Join(t.src, "bar"))
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Check.
	b, err := ioutil.ReadFile(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))

	fi0, err := os.Stat(path.Join(t.dst, "foo"))
	AssertEq(nil, err)

	fi1, err := os.Stat(path.Join(t.dst, "foo"))
	AssertEq(nil, err)

	ExpectTrue(os.SameFile(fi0, fi1))
}

func (t *SaveAndRestoreTest) Symlinks() {
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

func (t *SaveAndRestoreTest) ExistingScoreCaching() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) FileToScoreCaching_EntryOutOfDate() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) FileToScoreCaching_CachedScorePresent() {
	AssertFalse(true, "TODO")
}

func (t *SaveAndRestoreTest) FileToScoreCaching_CachedScoreNotPresent() {
	AssertFalse(true, "TODO")
}

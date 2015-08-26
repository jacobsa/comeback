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
	"log"
	"math/rand"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/save"
	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/syncutil"
	"github.com/jacobsa/timeutil"
)

func TestIntegration(t *testing.T) { RunTests(t) }

func init() {
	syncutil.EnableInvariantChecking()
}

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

var gDiscardLogger = log.New(ioutil.Discard, "", 0)

type commonTest struct {
	ctx            context.Context
	bucket         gcs.Bucket
	exclusions     []*regexp.Regexp
	existingScores util.StringSet
}

func (t *commonTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.bucket = gcsfake.NewFakeBucket(timeutil.RealClock(), "")
	t.existingScores = util.NewStringSet()
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
	_, _, err = wiring.MakeRegistryAndCrypter(t.ctx, password, t.bucket)
	AssertEq(nil, err)

	// Using a different password to do the same should fail.
	_, _, err = wiring.MakeRegistryAndCrypter(t.ctx, wrongPassword, t.bucket)
	ExpectThat(err, Error(HasSubstr("password is incorrect")))

	// Ditto with the dir restorer.
	_, err = wiring.MakeDirRestorer(t.ctx, wrongPassword, t.bucket)
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
	// Create the crypter.
	_, crypter, err := wiring.MakeRegistryAndCrypter(t.ctx, password, t.bucket)
	if err != nil {
		err = fmt.Errorf("MakeRegistryAndCrypter: %v", err)
		return
	}

	// Create the blob store.
	bs, err := wiring.MakeBlobStore(t.bucket, crypter, t.existingScores)
	if err != nil {
		err = fmt.Errorf("MakeBlobStore: %v", err)
		return
	}

	// Save the source directory.
	score, err = save.Save(
		t.ctx,
		t.src,
		t.exclusions,
		state.NewScoreMap(),
		bs,
		timeutil.RealClock())

	if err != nil {
		err = fmt.Errorf("Save: %v", err)
		return
	}

	return
}

// Restore a backup with the given root listing into t.dst.
func (t *SaveAndRestoreTest) restore(score blob.Score) (err error) {
	// Create the dir restorer.
	dirRestorer, err := wiring.MakeDirRestorer(t.ctx, password, t.bucket)
	if err != nil {
		err = fmt.Errorf("MakeDirRestorer: %v", err)
		return
	}

	// Call it.
	err = dirRestorer.RestoreDirectory(t.ctx, score, t.dst, "")
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

func (t *SaveAndRestoreTest) SingleEmptyFile() {
	var fi os.FileInfo
	var err error

	// Create.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte{}, 0400)
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
	ExpectEq(0, fi.Size())
	ExpectFalse(fi.IsDir())

	// Read the file.
	b, err := ioutil.ReadFile(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq("", string(b))
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

	// The files should have the same contents.
	b, err := ioutil.ReadFile(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))

	b, err = ioutil.ReadFile(path.Join(t.dst, "bar"))
	AssertEq(nil, err)
	ExpectEq(contents, string(b))

	// But they should be independent.
	fi0, err := os.Stat(path.Join(t.dst, "foo"))
	AssertEq(nil, err)

	fi1, err := os.Stat(path.Join(t.dst, "bar"))
	AssertEq(nil, err)

	ExpectFalse(os.SameFile(fi0, fi1))
}

func (t *SaveAndRestoreTest) Symlinks() {
	const target = "/foo/bar/baz"
	var err error

	// Create.
	err = os.Symlink(target, path.Join(t.src, "foo"))
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Check lstat.
	fi, err := os.Lstat(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(os.ModeSymlink, fi.Mode()&os.ModeType)

	// Check Readlink.
	actualTarget, err := os.Readlink(path.Join(t.dst, "foo"))
	AssertEq(nil, err)
	ExpectEq(target, actualTarget)
}

func (t *SaveAndRestoreTest) Permissions() {
	// This test currently fails because of a known bug: we restore the
	// directory's permissions (killing write access) before we restore the file
	// within it.
	//
	// TODO(jacobsa): Re-enable this test when issue #21 is fixed.
	log.Println("SKIPPING TEST DUE TO KNOWN BUG")
	return

	const contents = "taco"

	var fi os.FileInfo
	var err error

	// Create a directory that we initially have write access to.
	err = os.Mkdir(path.Join(t.src, "foo"), 0740)
	AssertEq(nil, err)

	// Create a file within it.
	err = ioutil.WriteFile(path.Join(t.src, "foo/bar"), []byte(contents), 0540)
	AssertEq(nil, err)

	// Seal off the directory.
	err = os.Chmod(path.Join(t.src, "foo"), 0540)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Check permissions.
	fi, err = os.Stat(path.Join(t.dst, "foo"))
	ExpectEq(os.FileMode(0540)|os.ModeDir, fi.Mode())

	fi, err = os.Stat(path.Join(t.dst, "foo/bar"))
	ExpectEq(os.FileMode(0540), fi.Mode())
}

func (t *SaveAndRestoreTest) Mtime() {
	var fi os.FileInfo
	var err error

	// Create.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte{}, 0400)
	AssertEq(nil, err)

	expected := time.Date(2012, 8, 15, 22, 56, 0, 0, time.Local)
	err = os.Chtimes(path.Join(t.src, "foo"), time.Time{}, expected)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Stat.
	fi, err = os.Stat(path.Join(t.dst, "foo"))
	ExpectThat(fi.ModTime(), timeutil.TimeEq(expected))
}

func (t *SaveAndRestoreTest) BackupExclusions() {
	var fi os.FileInfo
	var err error

	// Set up two exclusions.
	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile(".*bad0.*"),
		regexp.MustCompile(".*bad1.*"),
	}

	// Create some content that should be excluded.
	err = ioutil.WriteFile(path.Join(t.src, "bad0"), []byte{}, 0400)
	AssertEq(nil, err)

	err = os.Mkdir(path.Join(t.src, "bad1"), 0700)
	AssertEq(nil, err)

	err = ioutil.WriteFile(path.Join(t.src, "bad1/blah"), []byte{}, 0400)
	AssertEq(nil, err)

	// And one file that should be backed up.
	err = ioutil.WriteFile(path.Join(t.src, "foo"), []byte{}, 0400)
	AssertEq(nil, err)

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Only the one file should have made it.
	entries, err := ioutil.ReadDir(t.dst)
	AssertEq(nil, err)
	AssertEq(1, len(entries))

	fi = entries[0]
	ExpectEq("foo", fi.Name())
}

func (t *SaveAndRestoreTest) ExistingScoreCaching() {
	var err error

	// Save a backup.
	score, err := t.save()
	AssertEq(nil, err)

	// The score should have been added to the set of existing scores.
	ExpectTrue(t.existingScores.Contains(score.Hex()))

	// Delete the underlying object, then attempt to save again. We should get
	// the same score.
	objectName := wiring.BlobObjectNamePrefix + score.Hex()
	err = t.bucket.DeleteObject(
		t.ctx,
		&gcs.DeleteObjectRequest{Name: objectName})

	AssertEq(nil, err)

	newScore, err := t.save()
	AssertEq(nil, err)
	ExpectEq(score, newScore)

	// No new object for this score should have been saved.
	_, err = gcsutil.ReadObject(t.ctx, t.bucket, objectName)
	ExpectThat(err, HasSameTypeAs(&gcs.NotFoundError{}))
}

func (t *SaveAndRestoreTest) IdenticalFileContents() {
	const contents = "taco"
	var err error

	// Create multiple files at various places in a directory hierarchy, all with
	// identical contents.
	AssertEq(nil, os.Mkdir(path.Join(t.src, "dir0"), 0700))
	AssertEq(nil, os.Mkdir(path.Join(t.src, "dir1"), 0700))
	AssertEq(nil, os.Mkdir(path.Join(t.src, "dir1/sub"), 0700))

	paths := []string{
		"foo",
		"bar",
		"dir0/baz",
		"dir1/qux",
		"dir1/sub/norf",
	}

	for _, p := range paths {
		err = ioutil.WriteFile(path.Join(t.src, p), []byte(contents), 0400)
		AssertEq(nil, err)
	}

	// Save and restore.
	score, err := t.save()
	AssertEq(nil, err)

	err = t.restore(score)
	AssertEq(nil, err)

	// Each file should have made it through.
	for _, p := range paths {
		b, err := ioutil.ReadFile(path.Join(t.dst, p))
		AssertEq(nil, err)
		ExpectEq(contents, string(b))
	}
}

func (t *SaveAndRestoreTest) IdenticalDirectoryContents() {
	var fi os.FileInfo
	var err error

	names := []string{"dir0", "dir1"}
	mtime := time.Date(2015, 4, 5, 2, 15, 0, 0, time.Local)

	// Create the first directory and populate it.
	err = os.Mkdir(path.Join(t.src, names[0]), 0700)
	AssertEq(nil, err)

	file0 := path.Join(t.src, names[0], "foo")
	err = ioutil.WriteFile(file0, []byte("blah"), 0400)
	AssertEq(nil, err)

	err = os.Chtimes(file0, mtime, mtime)
	AssertEq(nil, err)

	// Create the second directory and stick a hard link to the same file in it.
	err = os.Mkdir(path.Join(t.src, names[1]), 0700)
	AssertEq(nil, err)

	file1 := path.Join(t.src, names[1], "foo")
	err = os.Link(file0, file1)
	AssertEq(nil, err)

	// Save.
	score, err := t.save()
	AssertEq(nil, err)

	// Sanity check: if we got the directory contents perfectly identical, then
	// we should see only these blobs in the bucket:
	//
	//  1. A listing for the root directory.
	//  2. A listing shared by the two directories.
	//  3. The contents of the (identical) files.
	//
	listReq := &gcs.ListObjectsRequest{Prefix: wiring.BlobObjectNamePrefix}
	objects, _, err := gcsutil.ListAll(t.ctx, t.bucket, listReq)

	AssertEq(nil, err)
	AssertEq(3, len(objects))

	// Restore.
	err = t.restore(score)
	AssertEq(nil, err)

	// Read each directory.
	for _, name := range names {
		entries, err := ioutil.ReadDir(path.Join(t.dst, name))
		AssertEq(nil, err)
		AssertEq(1, len(entries))

		fi = entries[0]
		ExpectEq("foo", fi.Name())
		ExpectEq(4, fi.Size())
		ExpectFalse(fi.IsDir())
	}
}

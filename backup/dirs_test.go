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
	"errors"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/backup/mock"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/fs/mock"
	"github.com/jacobsa/comeback/repr"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"regexp"
	"testing"
)

func TestRegisterDirsTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func saveBlob(res *[]byte) oglemock.Action {
	f := func(b []byte) (blob.Score, error) {
		*res = b
		return nil, errors.New("foo")
	}

	return oglemock.Invoke(f)
}

type readCloser struct {
	closed bool
}

func (r *readCloser) Read(b []byte) (int, error) {
	panic("Shouldn't be called.")
}

func (r *readCloser) Close() error {
	if r.closed {
		panic("Close called twice.")
	}

	r.closed = true
	return nil
}

func makeEntry(name string, t fs.EntryType) *fs.DirectoryEntry {
	return &fs.DirectoryEntry{
		Type: t,
		Name: name,
	}
}

type DirectorySaverTest struct {
	blobStore    mock_blob.MockStore
	fileSystem   mock_fs.MockFileSystem
	fileSaver    mock_backup.MockFileSaver
	wrapped      mock_backup.MockDirectorySaver
	linkResolver mock_backup.MockLinkResolver

	dirSaver backup.DirectorySaver

	basePath   string
	relPath    string
	exclusions []*regexp.Regexp

	score    blob.Score
	err      error
}

func init() { RegisterTestSuite(&DirectorySaverTest{}) }

func (t *DirectorySaverTest) SetUp(i *TestInfo) {
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.fileSaver = mock_backup.NewMockFileSaver(i.MockController, "fileSaver")
	t.wrapped = mock_backup.NewMockDirectorySaver(i.MockController, "wrapped")
	t.linkResolver = mock_backup.NewMockLinkResolver(i.MockController, "resolver")

	t.dirSaver, _ = backup.NewNonRecursiveDirectorySaver(
		t.blobStore,
		t.fileSystem,
		t.fileSaver,
		t.wrapped,
		t.linkResolver,
	)
}

func (t *DirectorySaverTest) callSaver() {
	t.score, t.err = t.dirSaver.Save(t.basePath, t.relPath, t.exclusions)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirectorySaverTest) CallsReadDir() {
	t.basePath = "/taco"
	t.relPath = "burrito/enchilada"

	// ReadDir
	ExpectCall(t.fileSystem, "ReadDir")("/taco/burrito/enchilada").
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callSaver()
}

func (t *DirectorySaverTest) ReadDirReturnsError() {
	// ReadDir
	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Listing")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectorySaverTest) NoEntriesInDirectory() {
	// ReadDir
	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return([]*fs.DirectoryEntry{}, nil))

	// Blob store
	var blob []byte
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(saveBlob(&blob))

	// Call
	t.callSaver()

	AssertNe(nil, blob)
	entries, err := repr.Unmarshal(blob)
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *DirectorySaverTest) AllEntriesExcluded() {
	t.basePath = "/tortilla"
	t.relPath = "taco"

	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile(`taco/q`),
		regexp.MustCompile(`taco/b`),
		regexp.MustCompile(`taco/[t-w]`),
	}

	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("burrito", fs.TypeFile),
		makeEntry("queso", fs.TypeDirectory),
		makeEntry("tttttt", fs.TypeSymlink),
		makeEntry("uuuuuu", fs.TypeBlockDevice),
		makeEntry("vvvvvv", fs.TypeCharDevice),
		makeEntry("wwwwww", fs.TypeNamedPipe),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Blob store
	var blob []byte
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(saveBlob(&blob))

	// Call
	t.callSaver()

	AssertNe(nil, blob)
	entries, err := repr.Unmarshal(blob)
	AssertEq(nil, err)
	ExpectThat(entries, ElementsAre())
}

func (t *DirectorySaverTest) CallsLinkResolverFileSystemAndFileSaverForFiles() {
	t.basePath = "/tortilla"
	t.relPath = "taco/queso"

	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile(`foobar`),  // Matches none
		regexp.MustCompile(`queso/n\w+s`),  // Matches second
		regexp.MustCompile(`bazqux`),  // Matches none
	}

	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("burrito", fs.TypeFile),
		makeEntry("nachos", fs.TypeFile),
		makeEntry("enchilada", fs.TypeFile),
	}

	entries[0].ContainingDevice = 17
	entries[0].Inode = 19

	entries[2].ContainingDevice = 23
	entries[2].Inode = 29

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Link resolver
	ExpectCall(t.linkResolver, "Register")(17, 19, "taco/queso/burrito").
		WillOnce(oglemock.Return(nil))

	ExpectCall(t.linkResolver, "Register")(23, 29, "taco/queso/enchilada").
		WillOnce(oglemock.Return(nil))

	// OpenForReading
	file0 := &readCloser{}
	file1 := &readCloser{}

	ExpectCall(t.fileSystem, "OpenForReading")("/tortilla/taco/queso/burrito").
		WillOnce(oglemock.Return(file0, nil))

	ExpectCall(t.fileSystem, "OpenForReading")("/tortilla/taco/queso/enchilada").
		WillOnce(oglemock.Return(file1, nil))

	// File saver
	ExpectCall(t.fileSaver, "Save")(file0).
		WillOnce(oglemock.Return([]blob.Score{}, nil))

	ExpectCall(t.fileSaver, "Save")(file1).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callSaver()
}

func (t *DirectorySaverTest) FileSystemReturnsErrorForOneFile() {
	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("", fs.TypeFile),
		makeEntry("", fs.TypeFile),
		makeEntry("", fs.TypeFile),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Link resolver
	ExpectCall(t.linkResolver, "Register")(Any(), Any(), Any()).
		WillRepeatedly(oglemock.Return(nil))

	// OpenForReading
	file0 := &readCloser{}

	ExpectCall(t.fileSystem, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(file0, nil)).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// File saver
	ExpectCall(t.fileSaver, "Save")(Any()).
		WillOnce(oglemock.Return([]blob.Score{}, nil))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Opening")))
	ExpectThat(t.err, Error(HasSubstr("taco")))

	ExpectTrue(file0.closed)
}

func (t *DirectorySaverTest) FileSaverReturnsErrorForOneFile() {
	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("", fs.TypeFile),
		makeEntry("", fs.TypeFile),
		makeEntry("", fs.TypeFile),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Link resolver
	ExpectCall(t.linkResolver, "Register")(Any(), Any(), Any()).
		WillRepeatedly(oglemock.Return(nil))

	// OpenForReading
	file0 := &readCloser{}
	file1 := &readCloser{}

	ExpectCall(t.fileSystem, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(file0, nil)).
		WillOnce(oglemock.Return(file1, nil))

	// File saver
	ExpectCall(t.fileSaver, "Save")(Any()).
		WillOnce(oglemock.Return([]blob.Score{}, nil)).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("taco")))

	ExpectTrue(file0.closed)
	ExpectTrue(file1.closed)
}

func (t *DirectorySaverTest) CallsDirSaverForDirs() {
	t.basePath = "/taco"
	t.relPath = "queso/tortilla"

	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile(`foobar`),  // Matches none
		regexp.MustCompile(`tortilla/n\w+s`),  // Matches second
		regexp.MustCompile(`bazqux`),  // Matches none
	}

	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("burrito", fs.TypeDirectory),
		makeEntry("nachos", fs.TypeDirectory),
		makeEntry("enchilada", fs.TypeDirectory),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Wrapped directory saver
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.wrapped, "Save")(
		"/taco",
		"queso/tortilla/burrito",
		DeepEquals(t.exclusions)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.wrapped, "Save")(
		"/taco",
		"queso/tortilla/enchilada",
		DeepEquals(t.exclusions)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callSaver()
}

func (t *DirectorySaverTest) DirSaverReturnsErrorForOneDir() {
	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("", fs.TypeDirectory),
		makeEntry("", fs.TypeDirectory),
		makeEntry("", fs.TypeDirectory),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Wrapped directory saver
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.wrapped, "Save")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(score0, nil)).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectorySaverTest) OneTypeIsUnsupported() {
	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("", fs.TypeDirectory),
		makeEntry("", fs.TypeDirectory),
		makeEntry("", fs.TypeDirectory),
	}

	entries[1].Type = 17

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Wrapped directory saver
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.wrapped, "Save")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(score0, nil))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Unhandled")))
	ExpectThat(t.err, Error(HasSubstr("type")))
}

func (t *DirectorySaverTest) CallsBlobStore() {
	t.exclusions = []*regexp.Regexp{
		regexp.MustCompile(`cilantro`),  // Matches third
	}

	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("taco", fs.TypeFile),
		makeEntry("burrito", fs.TypeDirectory),
		makeEntry("cilantro", fs.TypeNamedPipe),
		makeEntry("enchilada", fs.TypeDirectory),
		makeEntry("carnitas", fs.TypeSymlink),
		makeEntry("queso", fs.TypeBlockDevice),
		makeEntry("tortilla", fs.TypeCharDevice),
		makeEntry("nachos", fs.TypeNamedPipe),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Link resolver
	ExpectCall(t.linkResolver, "Register")(Any(), Any(), Any()).
		WillRepeatedly(oglemock.Return(nil))

	// OpenForReading
	file0 := &readCloser{}

	ExpectCall(t.fileSystem, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(file0, nil))

	// File saver
	score0 := blob.ComputeScore([]byte("nachos"))
	score1 := blob.ComputeScore([]byte("carnitas"))

	ExpectCall(t.fileSaver, "Save")(Any()).
		WillOnce(oglemock.Return([]blob.Score{score0, score1}, nil))

	// Wrapped directory saver
	score2 := blob.ComputeScore([]byte("queso"))
	score3 := blob.ComputeScore([]byte("tortilla"))

	ExpectCall(t.wrapped, "Save")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(score2, nil)).
		WillOnce(oglemock.Return(score3, nil))

	// Blob store
	var blob []byte
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(saveBlob(&blob))

	// Call
	t.callSaver()

	AssertNe(nil, blob, "Saver error: %v", t.err)
	resultEntries, err := repr.Unmarshal(blob)
	AssertEq(nil, err)
	AssertEq(7, len(resultEntries))

	entry := resultEntries[0]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("taco", entry.Name)
	AssertThat(entry.Scores, ElementsAre(Any(), Any()))
	ExpectThat(entry.Scores[0], DeepEquals(score0))
	ExpectThat(entry.Scores[1], DeepEquals(score1))
	ExpectEq(nil, entry.HardLinkTarget)

	entry = resultEntries[1]
	ExpectEq(fs.TypeDirectory, entry.Type)
	ExpectEq("burrito", entry.Name)
	AssertThat(entry.Scores, ElementsAre(Any()))
	ExpectThat(entry.Scores[0], DeepEquals(score2))
	ExpectEq(nil, entry.HardLinkTarget)

	entry = resultEntries[2]
	ExpectEq(fs.TypeDirectory, entry.Type)
	ExpectEq("enchilada", entry.Name)
	AssertThat(entry.Scores, ElementsAre(Any()))
	ExpectThat(entry.Scores[0], DeepEquals(score3))
	ExpectEq(nil, entry.HardLinkTarget)

	entry = resultEntries[3]
	ExpectEq(fs.TypeSymlink, entry.Type)
	ExpectEq("carnitas", entry.Name)

	entry = resultEntries[4]
	ExpectEq(fs.TypeBlockDevice, entry.Type)
	ExpectEq("queso", entry.Name)

	entry = resultEntries[5]
	ExpectEq(fs.TypeCharDevice, entry.Type)
	ExpectEq("tortilla", entry.Name)

	entry = resultEntries[6]
	ExpectEq(fs.TypeNamedPipe, entry.Type)
	ExpectEq("nachos", entry.Name)
}

func (t *DirectorySaverTest) FilesAreHardLinked() {
	// ReadDir
	entries := []*fs.DirectoryEntry{
		makeEntry("taco", fs.TypeFile),
		makeEntry("burrito", fs.TypeFile),
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Link resolver
	target0 := "/enchilada"
	target1 := "/queso"

	ExpectCall(t.linkResolver, "Register")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(&target0)).
		WillOnce(oglemock.Return(&target1))

	// Blob store
	var blob []byte
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(saveBlob(&blob))

	// Call
	t.callSaver()

	AssertNe(nil, blob, "Saver error: %v", t.err)
	resultEntries, err := repr.Unmarshal(blob)
	AssertEq(nil, err)
	AssertEq(2, len(resultEntries))

	entry := resultEntries[0]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("taco", entry.Name)
	AssertThat(entry.Scores, ElementsAre())
	ExpectThat(entry.HardLinkTarget, Pointee(Equals(target0)))

	entry = resultEntries[1]
	ExpectEq(fs.TypeFile, entry.Type)
	ExpectEq("burrito", entry.Name)
	AssertThat(entry.Scores, ElementsAre())
	ExpectThat(entry.HardLinkTarget, Pointee(Equals(target1)))
}

func (t *DirectorySaverTest) BlobStoreReturnsError() {
	// ReadDir
	entries := []*fs.DirectoryEntry{}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Blob store
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Storing")))
	ExpectThat(t.err, Error(HasSubstr("blob")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectorySaverTest) BlobStoreSucceeds() {
	// ReadDir
	entries := []*fs.DirectoryEntry{}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// Blob store
	score := blob.ComputeScore([]byte("hello"))
	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(score, nil))

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(t.score, DeepEquals(score))
}

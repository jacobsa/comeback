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
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
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

type DirectorySaverTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem
	fileSaver mock_backup.MockFileSaver
	wrapped mock_backup.MockDirectorySaver

	dirSaver backup.DirectorySaver

	dirpath string
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

func (t *DirectorySaverTest) callSaver() {
	t.score, t.err = t.dirSaver.Save(t.dirpath)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DirectorySaverTest) CallsReadDir() {
	t.dirpath = "taco"

	// ReadDir
	ExpectCall(t.fileSystem, "ReadDir")("taco").
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

func (t *DirectorySaverTest) CallsFileSystemAndFileSaverForFiles() {
	t.dirpath = "/taco"

	// ReadDir
	entries := []*fs.DirectoryEntry {
		&fs.DirectoryEntry{Name: "burrito"},
		&fs.DirectoryEntry{Name: "enchilada"},
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

	// OpenForReading
	file0 := &readCloser{}
	file1 := &readCloser{}

	ExpectCall(t.fileSystem, "OpenForReading")("/taco/burrito").
		WillOnce(oglemock.Return(file0, nil))

	ExpectCall(t.fileSystem, "OpenForReading")("/taco/enchilada").
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
	entries := []*fs.DirectoryEntry {
		&fs.DirectoryEntry{},
		&fs.DirectoryEntry{},
		&fs.DirectoryEntry{},
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

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
}

func (t *DirectorySaverTest) FileSaverReturnsErrorForOneFile() {
	// ReadDir
	entries := []*fs.DirectoryEntry {
		&fs.DirectoryEntry{},
		&fs.DirectoryEntry{},
		&fs.DirectoryEntry{},
	}

	ExpectCall(t.fileSystem, "ReadDir")(Any()).
		WillOnce(oglemock.Return(entries, nil))

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
}

func (t *DirectorySaverTest) ClosesFilesOnError() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) ClosesFilesOnSuccess() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) CallsDirSaverForDirs() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) DirSaverReturnsErrorForOneDir() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) OneTypeIsUnsupported() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) CallsBlobStore() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) BlobStoreReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectorySaverTest) BlobStoreSucceeds() {
	ExpectEq("TODO", "")
}

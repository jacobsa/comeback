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
	"testing"
)

func TestDirectoryRestorer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func returnEntries(entries []*fs.DirectoryEntry) oglemock.Action {
	data, err := repr.Marshal(entries)
	AssertEq(nil, err)

	return oglemock.Return(data, nil)
}

func makeStrPtr(s string) *string {
	return &s
}

type DirectoryRestorerTest struct {
	blobStore    mock_blob.MockStore
	fileSystem   mock_fs.MockFileSystem
	fileRestorer mock_backup.MockFileRestorer
	wrapped      mock_backup.MockDirectoryRestorer

	dirRestorer backup.DirectoryRestorer

	score    blob.Score
	basePath string
	relPath  string

	err error
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

func (t *DirectoryRestorerTest) CallsBlobStore() {
	t.score = []byte("taco")

	// Blob store
	ExpectCall(t.blobStore, "Load")(DeepEquals(t.score)).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *DirectoryRestorerTest) BlobStoreReturnsError() {
	// Blob store
	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Load")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectoryRestorerTest) BlobStoreReturnsJunk() {
	// Blob store
	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(oglemock.Return([]byte("taco"), nil))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("invalid")))
	ExpectThat(t.err, Error(HasSubstr("data")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectoryRestorerTest) NoEntries() {
	// Blob store
	entries := []*fs.DirectoryEntry{}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// Call
	t.call()

	ExpectEq(nil, t.err)
}

func (t *DirectoryRestorerTest) FileEntry_CallsLinkForHardLink() {
	t.basePath = "/foo"
	t.relPath = "bar/baz"

	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Name:           "taco",
			Type:           fs.TypeFile,
			HardLinkTarget: makeStrPtr("burrito/enchilada"),
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// File system
	ExpectCall(t.fileSystem, "CreateHardLink")(
		"/foo/burrito/enchilada",
		"/foo/bar/baz/taco",
	).WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.call()
}

func (t *DirectoryRestorerTest) FileEntry_LinkReturnsError() {
	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type:           fs.TypeFile,
			HardLinkTarget: makeStrPtr(""),
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// File system
	ExpectCall(t.fileSystem, "CreateHardLink")(Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("CreateHardLink")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectoryRestorerTest) FileEntry_LinkSucceeds() {
	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type:           fs.TypeFile,
			HardLinkTarget: makeStrPtr(""),
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// File system
	ExpectCall(t.fileSystem, "CreateHardLink")(Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	t.call()

	ExpectEq(nil, t.err)
}

func (t *DirectoryRestorerTest) FileEntry_CallsRestoreFile() {
	t.basePath = "/foo"
	t.relPath = "bar/baz"

	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Name:           "taco",
			Type:           fs.TypeFile,
			Permissions: 0712,
			Scores: []blob.Score{
				blob.ComputeScore([]byte("burrito")),
				blob.ComputeScore([]byte("enchilada")),
			},
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// File restorer
	ExpectCall(t.fileRestorer, "RestoreFile")(
		DeepEquals(entries[0].Scores),
		"/foo/bar/baz/taco",
		0712,
	).WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.call()
}

func (t *DirectoryRestorerTest) FileEntry_RestoreFileReturnsError() {
	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type:           fs.TypeFile,
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// File restorer
	ExpectCall(t.fileRestorer, "RestoreFile")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("RestoreFile")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *DirectoryRestorerTest) DirEntry_ZeroScores() {
	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type:           fs.TypeDirectory,
			Scores: []blob.Score{},
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("directory")))
	ExpectThat(t.err, Error(HasSubstr("entry")))
	ExpectThat(t.err, Error(HasSubstr("exactly one")))
	ExpectThat(t.err, Error(HasSubstr("score")))
}

func (t *DirectoryRestorerTest) DirEntry_TwoScores() {
	// Blob store
	entries := []*fs.DirectoryEntry{
		&fs.DirectoryEntry{
			Type:           fs.TypeDirectory,
			Scores: []blob.Score{
				blob.ComputeScore([]byte("a")),
				blob.ComputeScore([]byte("b")),
			},
		},
	}

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(returnEntries(entries))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("directory")))
	ExpectThat(t.err, Error(HasSubstr("entry")))
	ExpectThat(t.err, Error(HasSubstr("exactly one")))
	ExpectThat(t.err, Error(HasSubstr("score")))
}

func (t *DirectoryRestorerTest) DirEntry_CallsMkdir() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) DirEntry_MkdirReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) DirEntry_CallsWrapped() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) DirEntry_WrappedReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) SymlinkEntry_CallsSymlink() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) SymlinkEntry_SymlinkReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) PipeEntry_CallsCreate() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) PipeEntry_CreateReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) BlockDevEntry_CallsCreate() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) BlockDevEntry_CreateReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) CharDevEntry_CallsCreate() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) CharDevEntry_CreateReturnsError() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) CallsChown() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) ChownReturnsErrorForOneEntry() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) CallsSetModTime() {
	ExpectEq("TODO", "")
	// NOTE: Not for devices (see restore.go)
}

func (t *DirectoryRestorerTest) SetModTimeReturnsErrorForOneEntry() {
	ExpectEq("TODO", "")
}

func (t *DirectoryRestorerTest) EverythingSucceeds() {
	ExpectEq("TODO", "")
}

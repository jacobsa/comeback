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
	"bytes"
	"errors"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"os"
	"testing"
)

func TestFileRestorer(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type closeSnoopingBuffer struct {
	bytes.Buffer

	closeError error
	closeCalled bool
}

func (b *closeSnoopingBuffer) Close() error {
	b.closeCalled = true
	return b.closeError
}

type errorWriteCloser struct {
	n int
}

func (w *errorWriteCloser) Close() error {
	return nil
}

func (w *errorWriteCloser) Write(p []byte) (n int, err error) {
	n = len(p)
	w.n -= n

	if w.n <= 0 {
		err = errors.New("errorWriteCloser simulated errors")
	}

	return
}

type FileRestorerTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem
	file closeSnoopingBuffer

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

func (t *FileRestorerTest) CallsCreateFile() {
	t.path = "taco"
	t.perms = 0612

	// File system
	ExpectCall(t.fileSystem, "CreateFile")("taco", 0612).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.call()
}

func (t *FileRestorerTest) CreateFileReturnsError() {
	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("CreateFile")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FileRestorerTest) NoBlobs() {
	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(&t.file, nil))

	// Call
	t.call()
	AssertEq(nil, t.err)

	ExpectTrue(t.file.closeCalled)
	ExpectThat(t.file.Bytes(), ElementsAre())
}

func (t *FileRestorerTest) CallsBlobStore() {
	t.scores = []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
		blob.ComputeScore([]byte("baz")),
	}

	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(&t.file, nil))

	// Blob store
	ExpectCall(t.blobStore, "Load")(DeepEquals(t.scores[0])).
		WillOnce(oglemock.Return([]byte{}, nil))

	ExpectCall(t.blobStore, "Load")(DeepEquals(t.scores[1])).
		WillOnce(oglemock.Return([]byte{}, nil))

	ExpectCall(t.blobStore, "Load")(DeepEquals(t.scores[2])).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Call
	t.call()
}

func (t *FileRestorerTest) BlobStoreReturnsErrorForOneCall() {
	t.scores = []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
		blob.ComputeScore([]byte("baz")),
	}

	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(&t.file, nil))

	// Blob store
	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil)).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.call()

	ExpectTrue(t.file.closeCalled)
	ExpectThat(t.err, Error(HasSubstr("Load")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FileRestorerTest) WriteReturnsErrorForOneCall() {
	t.scores = []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	// File system
	writeCloser := &errorWriteCloser{7}
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(writeCloser, nil))

	// Blob store
	blob0 := []byte("taco")
	blob1 := []byte("burrito")

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(oglemock.Return(blob0, nil)).
		WillOnce(oglemock.Return(blob1, nil))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Write")))
	ExpectThat(t.err, Error(HasSubstr("errorWriteCloser")))
}

func (t *FileRestorerTest) CloseReturnsError() {
	t.file.closeError = errors.New("taco")

	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(&t.file, nil))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Close")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FileRestorerTest) EverythingSucceeds() {
	t.scores = []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	// File system
	ExpectCall(t.fileSystem, "CreateFile")(Any(), Any()).
		WillOnce(oglemock.Return(&t.file, nil))

	// Blob store
	blob0 := []byte("taco")
	blob1 := []byte("burrito")

	ExpectCall(t.blobStore, "Load")(Any()).
		WillOnce(oglemock.Return(blob0, nil)).
		WillOnce(oglemock.Return(blob1, nil))

	// Call
	t.call()
	AssertEq(nil, t.err)

	AssertTrue(t.file.closeCalled)
	ExpectEq("tacoburrito", t.file.String())
}

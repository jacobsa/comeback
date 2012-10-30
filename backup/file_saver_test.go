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
	"github.com/jacobsa/comeback/concurrent"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io"
	"testing"
	"testing/iotest"
	"time"
)

const (
	chunkSize           = 1 << 14
	numFileSaverWorkers = 5
)

func TestRegisterFilesTest(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeChunk(char int) []byte {
	charStr := string(char)
	AssertEq(1, len(charStr), "Invalid character: %d", char)
	return bytes.Repeat([]byte(charStr), chunkSize)
}

func returnStoreError(err string) oglemock.Action {
	f := func(b []byte) (blob.Score, error) { return nil, errors.New(err) }
	return oglemock.Invoke(f)
}

type readCloser struct {
	reader io.Reader
	closed bool
}

func (r *readCloser) Read(b []byte) (int, error) {
	return r.reader.Read(b)
}

func (r *readCloser) Close() error {
	if r.closed {
		panic("Close called twice.")
	}

	r.closed = true
	return nil
}

type FileSaverTest struct {
	blobStore mock_blob.MockStore
	fileSystem mock_fs.MockFileSystem
	executor  concurrent.Executor
	fileSaver backup.FileSaver

	file readCloser

	path string
	scores []blob.Score
	err    error
}

func init() { RegisterTestSuite(&FileSaverTest{}) }

func (t *FileSaverTest) SetUp(i *TestInfo) {
	var err error

	// Dependencies
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.executor = concurrent.NewExecutor(numFileSaverWorkers)

	// Saver
	t.fileSaver, err = backup.NewFileSaver(
		t.blobStore,
		chunkSize,
		t.fileSystem,
		t.executor,
	)

	AssertEq(nil, err)

	// By default, return the configured reader.
	ExpectCall(t.fileSystem, "OpenForReading")(Any()).
		WillRepeatedly(oglemock.Return(&t.file, nil))
}

func (t *FileSaverTest) callSaver() {
	t.scores, t.err = t.fileSaver.SavePath(t.path)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSaverTest) ZeroChunkSize() {
	_, err := backup.NewFileSaver(t.blobStore, 0, t.fileSystem, t.executor)
	ExpectThat(err, Error(HasSubstr("size")))
	ExpectThat(err, Error(HasSubstr("positive")))
}

func (t *FileSaverTest) CallsOpenForReading() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) OpenForReadingReturnsError() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) NoDataInReader() {
	// Reader
	t.file.reader = new(bytes.Buffer)

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	AssertTrue(t.file.closed)
	ExpectThat(t.scores, ElementsAre())
}

func (t *FileSaverTest) ReadErrorInFirstChunk() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')

	// Reader
	t.file.reader = io.MultiReader(
		iotest.TimeoutReader(bytes.NewReader(chunk0)),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	ExpectCall(t.blobStore, "Store")(Any()).Times(0)

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Reading")))
	ExpectThat(t.err, Error(HasSubstr("chunk")))
	ExpectThat(t.err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *FileSaverTest) ReadErrorInMiddleChunk() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('b')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		iotest.TimeoutReader(bytes.NewReader(chunk1)),
		bytes.NewReader(chunk2),
	)

	// Blob store
	ExpectCall(t.blobStore, "Store")(Any()).Times(1)

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Reading")))
	ExpectThat(t.err, Error(HasSubstr("chunk")))
	ExpectThat(t.err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *FileSaverTest) CopesWithShortReadsWithinFullSizeChunks() {
	// Chunks
	chunk0 := makeChunk('a')

	// Reader
	t.file.reader = io.MultiReader(
		iotest.OneByteReader(bytes.NewReader(chunk0)),
	)

	// Blob store
	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(returnStoreError(""))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) CopesWithEofAndNonZeroData() {
	// Chunks
	chunk0 := makeChunk('a')

	// Reader
	t.file.reader = io.MultiReader(
		iotest.DataErrReader(bytes.NewReader(chunk0)),
	)

	// Blob store
	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(returnStoreError(""))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) OneSmallerSizedChunk() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk0 = chunk0[0 : len(chunk0)-10]

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) OneFullSizedChunk() {
	// Chunks
	chunk0 := makeChunk('a')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) OneFullSizedChunkPlusOneByte() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := []byte{0xde}

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))
	score1 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(oglemock.Return(score1, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) MultipleChunksWithNoRemainder() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))
	score1 := blob.ComputeScore([]byte(""))
	score2 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(oglemock.Return(score1, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk2)).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) MultipleChunksWithSmallRemainder() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := []byte{0xde, 0xad}

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))
	score1 := blob.ComputeScore([]byte(""))
	score2 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(oglemock.Return(score1, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk2)).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) MultipleChunksWithLargeRemainder() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')
	chunk2 = chunk2[0 : len(chunk2)-1]

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))
	score1 := blob.ComputeScore([]byte(""))
	score2 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(oglemock.Return(score1, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk2)).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()
}

func (t *FileSaverTest) ErrorStoringOneChunk() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))
	score2 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(score0, nil)).
		WillOnce(returnStoreError("taco")).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Storing")))
	ExpectThat(t.err, Error(HasSubstr("chunk")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FileSaverTest) ResultForEmptyReader() {
	// Reader
	t.file.reader = io.MultiReader()

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(t.scores, ElementsAre())
}

func (t *FileSaverTest) AllStoresSuccessful() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte("taco"))
	score1 := blob.ComputeScore([]byte("burrito"))
	score2 := blob.ComputeScore([]byte("enchilada"))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(oglemock.Return(score0, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(oglemock.Return(score1, nil))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk2)).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(
		t.scores,
		ElementsAre(
			DeepEquals(score0),
			DeepEquals(score1),
			DeepEquals(score2),
		),
	)
}

func (t *FileSaverTest) StoresFinishOutOfOrder() {
	AssertGt(numFileSaverWorkers, 1)

	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')
	chunk2 := makeChunk('c')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte("taco"))
	score1 := blob.ComputeScore([]byte("burrito"))
	score2 := blob.ComputeScore([]byte("enchilada"))

	sleepThenReturn := func(d time.Duration, s blob.Score) oglemock.Action {
		return oglemock.Invoke(func([]byte) (blob.Score, error) {
			time.Sleep(d)
			return s, nil
		})
	}

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk0)).
		WillOnce(sleepThenReturn(200*time.Millisecond, score0))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk1)).
		WillOnce(sleepThenReturn(100*time.Millisecond, score1))

	ExpectCall(t.blobStore, "Store")(DeepEquals(chunk2)).
		WillOnce(sleepThenReturn(150*time.Millisecond, score2))

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(
		t.scores,
		ElementsAre(
			DeepEquals(score0),
			DeepEquals(score1),
			DeepEquals(score2),
		),
	)
}

func (t *FileSaverTest) ClosesFile() {
	// Chunks
	chunk0 := makeChunk('a')

	// Reader
	t.file.reader = io.MultiReader(
		bytes.NewReader(chunk0),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte("taco"))

	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(score0, nil))

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectTrue(t.file.closed)
}

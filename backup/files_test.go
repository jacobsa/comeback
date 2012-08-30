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

package backup

import (
	"bytes"
	"errors"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io"
	"testing"
	"testing/iotest"
)

const (
	chunkSize = 1 << 14
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

type FileSaverTest struct {
	blobStore mock_blob.MockStore
	reader    io.Reader
	fileSaver FileSaver

	scores []blob.Score
	err    error
}

func init() { RegisterTestSuite(&FileSaverTest{}) }

func (t *FileSaverTest) SetUp(i *TestInfo) {
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.fileSaver, _ = NewFileSaver(t.blobStore, chunkSize)
}

func (t *FileSaverTest) callSaver() {
	t.scores, t.err = t.fileSaver.Save(t.reader)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSaverTest) ZeroChunkSize() {
	_, err := NewFileSaver(t.blobStore, 0)
	ExpectThat(err, Error(HasSubstr("size")))
	ExpectThat(err, Error(HasSubstr("positive")))
}

func (t *FileSaverTest) NoDataInReader() {
	// Reader
	t.reader = new(bytes.Buffer)

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(t.scores, ElementsAre())
}

func (t *FileSaverTest) ReadErrorInFirstChunk() {
	// Chunks
	chunk0 := makeChunk('a')

	// Reader
	t.reader = io.MultiReader(
		iotest.TimeoutReader(bytes.NewReader(chunk0)),
	)

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Reading")))
	ExpectThat(t.err, Error(HasSubstr("chunk")))
	ExpectThat(t.err, Error(HasSubstr(iotest.ErrTimeout.Error())))
}

func (t *FileSaverTest) ReadErrorInSecondChunk() {
	// Chunks
	chunk0 := makeChunk('a')
	chunk1 := makeChunk('b')

	// Reader
	t.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		iotest.TimeoutReader(bytes.NewReader(chunk1)),
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
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
	t.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte(""))

	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(score0, nil)).
		WillOnce(returnStoreError("taco"))

	// Call
	t.callSaver()

	ExpectThat(t.err, Error(HasSubstr("Storing")))
	ExpectThat(t.err, Error(HasSubstr("chunk")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FileSaverTest) ResultForEmptyReader() {
	// Reader
	t.reader = io.MultiReader()

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
	t.reader = io.MultiReader(
		bytes.NewReader(chunk0),
		bytes.NewReader(chunk1),
		bytes.NewReader(chunk2),
	)

	// Blob store
	score0 := blob.ComputeScore([]byte("taco"))
	score1 := blob.ComputeScore([]byte("burrito"))
	score2 := blob.ComputeScore([]byte("enchilada"))

	ExpectCall(t.blobStore, "Store")(Any()).
		WillOnce(oglemock.Return(score0, nil)).
		WillOnce(oglemock.Return(score1, nil)).
		WillOnce(oglemock.Return(score2, nil))

	// Call
	t.callSaver()

	AssertEq(nil, t.err)
	ExpectThat(t.scores, ElementsAre(score0, score1, score2))
}

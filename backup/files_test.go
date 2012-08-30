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
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"bytes"
	"errors"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/io/mock"
	"github.com/jacobsa/oglemock"
	"io"
	"testing"
	"testing/iotest"
)

const (
	expectedChunkSize = 1<<24
)

func TestRegister(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func makeChunk(char int) []byte {
	charStr := string(char)
	AssertEq(1, len(charStr), "Invalid character: %d", char)
	return bytes.Repeat([]byte(charStr), expectedChunkSize)
}

func returnStoreError(err string) oglemock.Action {
	f := func(b []byte) (blob.Score, error) { return nil, errors.New(err) }
	return oglemock.Invoke(f)
}

type FileSaverTest struct {
	blobStore mock_blob.MockStore
	mockReader mock_io.MockReader
	fileSaver FileSaver

	// Will be used instead of mockReader if non-nil.
	reader io.Reader

	scores []blob.Score
	err error
}

func init() { RegisterTestSuite(&FileSaverTest{}) }

func (t *FileSaverTest) SetUp(i *TestInfo) {
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.mockReader = mock_io.NewMockReader(i.MockController, "reader")
	t.fileSaver, _ = NewFileSaver(t.blobStore)
}

func (t *FileSaverTest) callSaver() {
	var reader io.Reader = t.mockReader
	if t.reader != nil {
		reader = t.reader
	}

	t.scores, t.err = t.fileSaver.Save(reader)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

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
	chunk0 = chunk0[0:len(chunk0)-10]

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
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) MultipleChunksWithNoRemainder() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) MultipleChunksWithSmallRemainder() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) MultipleChunksWithLargeRemainder() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) ErrorStoringOneChunk() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) ResultForEmptyReader() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) AllStoresSuccessful() {
	ExpectEq("TODO", "")
}

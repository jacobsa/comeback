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
	. "github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/mock"
	"github.com/jacobsa/comeback/io/mock"
	"testing"
)

const (
	expectedChunkSize = 1<<24
)

func TestRegister(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type FileSaverTest struct {
	blobStore mock_blob.MockStore
	reader mock_io.MockReader
	fileSaver FileSaver

	scores []blob.Score
	err error
}

func init() { RegisterTestSuite(&FileSaverTest{}) }

func (t *FileSaverTest) SetUp(i *TestInfo) {
	t.blobStore = mock_blob.NewMockStore(i.MockController, "blobStore")
	t.reader = mock_io.NewMockReader(i.MockController, "reader")
	t.fileSaver, _ = NewFileSaver(t.blobStore)
}

func (t *FileSaverTest) callSaver() {
	t.scores, t.err = t.fileSaver.Save(t.reader)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *FileSaverTest) NoDataInReader() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) ImmediateReadError() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) ReadErrorInFirstChunk() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) ReadErrorInSecondChunk() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) CopesWithShortReads() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) CopesWithEofAndZeroData() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) CopesWithEofAndNonZeroData() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) OneSmallerSizedChunk() {
	ExpectEq("TODO", "")
}

func (t *FileSaverTest) OneFullSizedChunk() {
	ExpectEq("TODO", "")
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

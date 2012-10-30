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

package state_test

import (
	"errors"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/backup/mock"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/fs/mock"
	"github.com/jacobsa/comeback/state"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
	"time"
)

func TestMapReadingSaver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type MapReadingSaverTest struct {
	scoreMap   state.ScoreMap
	fileSystem mock_fs.MockFileSystem
	wrapped    mock_backup.MockFileSaver
	saver      backup.FileSaver

	path   string
	scores []blob.Score
	err    error
}

func init() { RegisterTestSuite(&MapReadingSaverTest{}) }

func (t *MapReadingSaverTest) SetUp(i *TestInfo) {
	t.scoreMap = state.NewScoreMap()
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.wrapped = mock_backup.NewMockFileSaver(i.MockController, "wrapped")
	t.saver = state.NewMapReadingFileSaver(t.scoreMap, t.fileSystem, t.wrapped)
}

func (t *MapReadingSaverTest) call() {
	t.scores, t.err = t.saver.Save(t.path)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MapReadingSaverTest) CallsStat() {
	t.path = "taco"

	// File system
	ExpectCall(t.fileSystem, "Stat")("taco").
		WillOnce(oglemock.Return(fs.DirectoryEntry{}, errors.New("")))

	// Call
	t.call()
}

func (t *MapReadingSaverTest) StatReturnsError() {
	// File system
	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(fs.DirectoryEntry{}, errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Stat")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *MapReadingSaverTest) ScoreMapContainsEntry() {
	t.path = "taco"

	// Score map
	expectedKey := state.ScoreMapKey{
		Path: "taco",
		Permissions: 0644,
		Uid: 17,
		Gid: 19,
		MTime: time.Now(),
		Inode: 23,
		Size: 29,
	}

	expectedScores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	t.scoreMap.Set(expectedKey, expectedScores)

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid: 17,
		Gid: 19,
		MTime: time.Now(),
		Inode: 23,
		Size: 29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Call
	t.call()

	AssertEq(nil, t.err)
	ExpectThat(t.scores, DeepEquals(expectedScores))
}

func (t *MapReadingSaverTest) CallsWrapped() {
	t.path = "taco"

	// File system
	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(fs.DirectoryEntry{}, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Save")("taco").
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.call()
}

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

package state

import (
	"errors"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/backup/mock"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
	"time"
)

func TestScoreMapSaver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ScoreMapSaverTest struct {
	sourceMap  ScoreMap
	sinkMap    ScoreMap
	fileSystem mock_fs.MockFileSystem
	wrapped    mock_backup.MockFileSaver
	now        time.Time
	nowFunc    func () time.Time

	saver      backup.FileSaver

	path   string
	scores []blob.Score
	err    error
}

func init() { RegisterTestSuite(&ScoreMapSaverTest{}) }

func (t *ScoreMapSaverTest) SetUp(i *TestInfo) {
	t.sourceMap = NewScoreMap()
	t.sinkMap = NewScoreMap()
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.wrapped = mock_backup.NewMockFileSaver(i.MockController, "wrapped")
	t.now = time.Date(2012, time.August, 15, 22, 56, 00, 00, time.Local)
	t.nowFunc = func () time.Time { return t.now }

	t.saver = newScoreMapFileSaver(
		t.sourceMap,
		t.sinkMap,
		t.fileSystem,
		t.wrapped,
		t.nowFunc,
	)
}

func (t *ScoreMapSaverTest) call() {
	t.scores, t.err = t.saver.Save(t.path)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ScoreMapSaverTest) CallsStat() {
	t.path = "taco"

	// File system
	ExpectCall(t.fileSystem, "Stat")("taco").
		WillOnce(oglemock.Return(fs.DirectoryEntry{}, errors.New("")))

	// Call
	t.call()
}

func (t *ScoreMapSaverTest) StatReturnsError() {
	// File system
	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(fs.DirectoryEntry{}, errors.New("taco")))

	// Call
	t.call()

	ExpectThat(t.err, Error(HasSubstr("Stat")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *ScoreMapSaverTest) ScoreMapContainsEntry() {
	t.path = "taco"

	expectedKey := ScoreMapKey{
		Path:        "taco",
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       t.now.Add(-15 * time.Minute),
		Inode:       23,
		Size:        29,
	}

	// Source map
	expectedScores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	t.sourceMap.Set(expectedKey, expectedScores)

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       expectedKey.MTime,
		Inode:       23,
		Size:        29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Call
	t.call()

	AssertEq(nil, t.err)
	ExpectThat(t.scores, DeepEquals(expectedScores))
	ExpectThat(t.sinkMap.Get(expectedKey), DeepEquals(expectedScores))
}

func (t *ScoreMapSaverTest) CallsWrapped() {
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

func (t *ScoreMapSaverTest) WrappedReturnsError() {
	expectedKey := ScoreMapKey{
		Path:        "taco",
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       t.now.Add(-15 * time.Minute),
		Inode:       23,
		Size:        29,
	}

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       expectedKey.MTime,
		Inode:       23,
		Size:        29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Wrapped
	ExpectCall(t.wrapped, "Save")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.call()

	AssertThat(t.err, Error(Equals("taco")))
	ExpectEq(nil, t.sinkMap.Get(expectedKey))
}

func (t *ScoreMapSaverTest) WrappedReturnsScores() {
	t.path = "taco"
	expectedKey := ScoreMapKey{
		Path:        "taco",
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       t.now.Add(-15 * time.Minute),
		Inode:       23,
		Size:        29,
	}

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       expectedKey.MTime,
		Inode:       23,
		Size:        29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Wrapped
	expectedScores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
		blob.ComputeScore([]byte("bar")),
	}

	ExpectCall(t.wrapped, "Save")(Any()).
		WillOnce(oglemock.Return(expectedScores, nil))

	// Call
	t.call()

	AssertEq(nil, t.err)
	ExpectThat(t.sinkMap.Get(expectedKey), DeepEquals(expectedScores))
}

func (t *ScoreMapSaverTest) MTimeInFuture() {
	t.path = "taco"
	mapKey := ScoreMapKey{
		Path:        "taco",
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       t.now.Add(time.Minute),
		Inode:       23,
		Size:        29,
	}

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       mapKey.MTime,
		Inode:       23,
		Size:        29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Wrapped
	scores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
	}

	ExpectCall(t.wrapped, "Save")(Any()).
		WillOnce(oglemock.Return(scores, nil))

	// Call
	t.call()

	AssertEq(nil, t.err)
	ExpectEq(nil, t.sinkMap.Get(mapKey))
}

func (t *ScoreMapSaverTest) MTimeInRecentPast() {
	t.path = "taco"
	mapKey := ScoreMapKey{
		Path:        "taco",
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       t.now.Add(-30 * time.Second),
		Inode:       23,
		Size:        29,
	}

	// File system
	entry := fs.DirectoryEntry{
		Permissions: 0644,
		Uid:         17,
		Gid:         19,
		MTime:       mapKey.MTime,
		Inode:       23,
		Size:        29,
	}

	ExpectCall(t.fileSystem, "Stat")(Any()).
		WillOnce(oglemock.Return(entry, nil))

	// Wrapped
	scores := []blob.Score{
		blob.ComputeScore([]byte("foo")),
	}

	ExpectCall(t.wrapped, "Save")(Any()).
		WillOnce(oglemock.Return(scores, nil))

	// Call
	t.call()

	AssertEq(nil, t.err)
	ExpectEq(nil, t.sinkMap.Get(mapKey))
}

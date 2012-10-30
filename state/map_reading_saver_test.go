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
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/kv/mock"
	"github.com/jacobsa/comeback/state"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestMapReadingSaver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type MapReadingSaverTest struct {
	scoreMap state.ScoreMap
	fileSystem      mock_fs.MockFileSystem
	wrapped      mock_backup.MockFileSaver
	saver        backup.FileSaver

	path string
	scores []blob.Score
}

func init() { RegisterTestSuite(&MapReadingSaverTest{}) }

func (t *MapReadingSaverTest) SetUp(i *TestInfo) {
	t.scoreMap = state.NewScoreMap()
	t.fileSystem = mock_fs.NewMockFileSystem(i.MockController, "fileSystem")
	t.wrapped = mock_backup.NewMockFileSaver(i.MockController, "wrapped")
	t.saver = state.NewMapReadingFileSaver(t.scores, t.fileSystem, t.wrapped)
}

func (t *MapReadingSaverTest) call()

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *MapReadingSaverTest) DoesFoo() {
	ExpectEq("TODO", "")
}

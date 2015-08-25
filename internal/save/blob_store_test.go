// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package save

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/blob/mock"
	. "github.com/jacobsa/ogletest"
)

func TestBlobStore(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx       context.Context
	blobStore blob.Store

	node fsNode
}

func init() { RegisterTestSuite(&VisitorTest{}) }

func (t *VisitorTest) SetUp(ti *TestInfo) {
	t.ctx = ti.Ctx
	t.blobStore = mock_blob.NewMockStore(ti.MockController, "blobStore")
}

func (t *VisitorTest) call() (err error) {
	visitor := newVisitor(t.blobStore, make(chan *fsNode, 1))
	err = visitor.Visit(t.ctx, &t.node)
	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) ScoresAlreadyPresent_Empty() {
	scores := []blob.Score{}
	t.node.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectEq(scores, t.node.Scores)
}

func (t *VisitorTest) ScoresAlreadyPresent_NonEmpty() {
	scores := []blob.Score{blob.ComputeScore([]byte("taco"))}
	t.node.Scores = scores

	err := t.call()
	AssertEq(nil, err)
	ExpectEq(scores, t.node.Scores)
}

func (t *VisitorTest) Symlink() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) Directory() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_Empty() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsFull() {
	AssertTrue(false, "TODO")
}

func (t *VisitorTest) File_LastChunkIsPartial() {
	AssertTrue(false, "TODO")
}

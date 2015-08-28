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

package restore

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	. "github.com/jacobsa/ogletest"
)

func TestVisitor(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type VisitorTest struct {
	ctx       context.Context
	blobStore blob.Store

	// A directory that is deleted when the test completes.
	dir string

	// A visitor configured with the above directory.
	visitor dag.Visitor
}

var _ SetUpInterface = &VisitorTest{}
var _ TearDownInterface = &VisitorTest{}

func init() { RegisterTestSuite(&VisitorTest{}) }

func (t *VisitorTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Create the blob store.
	t.blobStore, err = newFakeBlobStore(t.ctx)
	AssertEq(nil, err)

	// Set up the directory.
	t.dir, err = ioutil.TempDir("", "visitor_test")
	AssertEq(nil, err)

	// Create the visitor.
	t.visitor = newVisitor(t.dir, t.blobStore, log.New(ioutil.Discard, "", 0))
}

func (t *VisitorTest) TearDown() {
	var err error

	err = os.RemoveAll(t.dir)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *VisitorTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

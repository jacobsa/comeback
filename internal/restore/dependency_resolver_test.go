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
	"testing"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

func TestDependencyResolver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func convertNodes(dagNodes []dag.Node) (nodes []*node) {
	for _, n := range dagNodes {
		nodes = append(nodes, n.(*node))
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type DependencyResolverTest struct {
	ctx       context.Context
	blobStore blob.Store
	dr        dag.DependencyResolver
}

var _ SetUpInterface = &DependencyResolverTest{}

func init() { RegisterTestSuite(&DependencyResolverTest{}) }

func (t *DependencyResolverTest) SetUp(ti *TestInfo) {
	var err error
	t.ctx = ti.Ctx

	// Create the blob store.
	bucket := gcsfake.NewFakeBucket(timeutil.RealClock(), "some_bucket")

	_, crypter, err := wiring.MakeRegistryAndCrypter(t.ctx, "password", bucket)
	AssertEq(nil, err)

	t.blobStore, err = wiring.MakeBlobStore(bucket, crypter, util.NewStringSet())
	AssertEq(nil, err)

	// Create the dependency resolver.
	t.dr = newDependencyResolver(t.blobStore, log.New(ioutil.Discard, "", 0))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DependencyResolverTest) DoesFoo() {
	AssertTrue(false, "TODO")
}

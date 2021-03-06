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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/gcloud/gcs/gcsfake"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"github.com/jacobsa/timeutil"
)

const (
	objectNamePrefix = "blobs/"
)

func TestDependencyResolver(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func newFakeBlobStore(ctx context.Context) (blobStore blob.Store, err error) {
	// Create a bucket.
	bucket := gcsfake.NewFakeBucket(timeutil.RealClock(), "some_bucket")

	// And a cryptoer.
	_, crypter, err := wiring.MakeRegistryAndCrypter(ctx, "password", bucket)
	if err != nil {
		err = fmt.Errorf("MakeRegistryAndCrypter: %v", err)
		return
	}

	// And the blob store.
	blobStore = newBlobStore(bucket, objectNamePrefix, crypter)
	if err != nil {
		err = fmt.Errorf("MakeBlobStore: %v", err)
		return
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
	t.blobStore, err = newFakeBlobStore(t.ctx)
	AssertEq(nil, err)

	// Create the dependency resolver.
	t.dr = newDependencyResolver(t.blobStore, log.New(ioutil.Discard, "", 0))
}

func (t *DependencyResolverTest) call(n *node) (deps []*node, err error) {
	untyped, err := t.dr.FindDependencies(t.ctx, n)
	if err != nil {
		err = fmt.Errorf("FindDependencies: %v", err)
		return
	}

	for _, u := range untyped {
		deps = append(deps, u.(*node))
	}

	return
}

func (t *DependencyResolverTest) store(b []byte) (s blob.Score, err error) {
	s, err = t.blobStore.Save(
		t.ctx,
		&blob.SaveRequest{Blob: b})

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *DependencyResolverTest) File() {
	n := &node{
		Info: fs.FileInfo{
			Type: fs.TypeFile,
		},
	}

	// Call
	deps, err := t.call(n)

	AssertEq(nil, err)
	ExpectThat(deps, ElementsAre())
	ExpectThat(n.Children, ElementsAre())
}

func (t *DependencyResolverTest) Symlink() {
	n := &node{
		Info: fs.FileInfo{
			Type: fs.TypeSymlink,
		},
	}

	// Call
	deps, err := t.call(n)

	AssertEq(nil, err)
	ExpectThat(deps, ElementsAre())
	ExpectThat(n.Children, ElementsAre())
}

func (t *DependencyResolverTest) BlobMissing() {
	s := blob.ComputeScore([]byte(""))
	n := &node{
		Info: fs.FileInfo{
			Type:   fs.TypeDirectory,
			Scores: []blob.Score{s},
		},
	}

	// Call
	_, err := t.call(n)

	ExpectThat(err, Error(HasSubstr("not found")))
	ExpectThat(err, Error(HasSubstr(s.Hex())))
}

func (t *DependencyResolverTest) BlobCorrupted() {
	var err error

	// Store some junk and set up a node with the junk's score as its contents.
	junk, err := t.store([]byte("foobar"))
	AssertEq(nil, err)

	n := &node{
		Info: fs.FileInfo{
			Type:   fs.TypeDirectory,
			Scores: []blob.Score{junk},
		},
	}

	// Call
	_, err = t.call(n)

	ExpectThat(err, Error(HasSubstr("UnmarshalDir")))
	ExpectThat(err, Error(HasSubstr(junk.Hex())))
}

func (t *DependencyResolverTest) NoChildren() {
	var err error

	// Set up an empty listing.
	listing := []*fs.FileInfo{}

	serialized, err := repr.MarshalDir(listing)
	AssertEq(nil, err)

	score, err := t.store(serialized)
	AssertEq(nil, err)

	// Set up the node.
	n := &node{
		RelPath: "taco/burrito",
		Info: fs.FileInfo{
			Type:   fs.TypeDirectory,
			Scores: []blob.Score{score},
		},
	}

	// Call
	deps, err := t.call(n)

	AssertEq(nil, err)
	ExpectThat(deps, ElementsAre())
	ExpectThat(n.Children, ElementsAre())
}

func (t *DependencyResolverTest) SomeChildren() {
	var err error

	// Set up a listing.
	listing := []*fs.FileInfo{
		&fs.FileInfo{
			Type:        fs.TypeFile,
			Name:        "foo",
			Permissions: 0754,
			MTime:       time.Now().Round(time.Millisecond),
		},
		&fs.FileInfo{
			Type:   fs.TypeDirectory,
			Name:   "bar",
			Scores: []blob.Score{blob.ComputeScore([]byte(""))},
			MTime:  time.Now().Round(time.Millisecond),
		},
	}

	serialized, err := repr.MarshalDir(listing)
	AssertEq(nil, err)

	score, err := t.store(serialized)
	AssertEq(nil, err)

	// Set up the node.
	n := &node{
		RelPath: "taco/burrito",
		Info: fs.FileInfo{
			Type:   fs.TypeDirectory,
			Scores: []blob.Score{score},
		},
	}

	// Call
	deps, err := t.call(n)

	AssertEq(nil, err)
	AssertEq(2, len(deps))
	AssertThat(n.Children, DeepEquals(deps))
	var child *node

	child = n.Children[0]
	ExpectEq("taco/burrito/foo", child.RelPath)
	ExpectThat(child.Info, DeepEquals(*listing[0]))

	child = n.Children[1]
	ExpectEq("taco/burrito/bar", child.RelPath)
	ExpectThat(child.Info, DeepEquals(*listing[1]))
}

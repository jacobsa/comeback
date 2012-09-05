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

package disk_test

import (
	"bytes"
	"errors"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/blob/disk"
	"github.com/jacobsa/comeback/fs/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io"
	"path"
	"testing"
)

func TestDisk(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type fakeFile struct {
	r      io.Reader
	closed bool
}

func (f *fakeFile) Read(p []byte) (int, error) {
	return f.r.Read(p)
}

func (f *fakeFile) Close() error {
	if f.closed {
		panic("Close called twice.")
	}

	f.closed = true
	return nil
}

type diskStoreTest struct {
	basePath string
	fs       mock_fs.MockFileSystem
	store    blob.Store
}

func (t *diskStoreTest) SetUp(i *TestInfo) {
	var err error

	t.basePath = "/foo/bar"
	t.fs = mock_fs.NewMockFileSystem(i.MockController, "fs")

	t.store, err = disk.NewDiskBlobStore(t.basePath, t.fs)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

type StoreTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&StoreTest{}) }

func (t *StoreTest) CallsFileSystem() {
	b := []byte("taco")

	// File system
	expectedPath := path.Join(t.basePath, "9dc4319c27f6479adc842ebef4a324a40759b95c")
	ExpectCall(t.fs, "WriteFile")(expectedPath, DeepEquals(b), 0600).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.store.Store(b)
}

func (t *StoreTest) FileSystemReturnsError() {
	// File system
	ExpectCall(t.fs, "WriteFile")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	_, err := t.store.Store([]byte{})

	ExpectThat(err, Error(HasSubstr("WriteFile")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *StoreTest) FileSystemSaysOkay() {
	b := []byte("taco")

	// File system
	ExpectCall(t.fs, "WriteFile")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	score, err := t.store.Store([]byte{})
	AssertEq(nil, err)

	ExpectThat(score, DeepEquals(blob.ComputeScore(b)))
}

////////////////////////////////////////////////////////////////////////
// Load
////////////////////////////////////////////////////////////////////////

type LoadTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&LoadTest{}) }

func (t *LoadTest) CallsFileSystem() {
	score := blob.ComputeScore([]byte("taco"))

	// File system
	expectedPath := path.Join(t.basePath, "9dc4319c27f6479adc842ebef4a324a40759b95c")
	ExpectCall(t.fs, "OpenForReading")(expectedPath).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Load(score)
}

func (t *LoadTest) FileSystemReturnsError() {
	// File system
	ExpectCall(t.fs, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Load(blob.ComputeScore([]byte{}))

	ExpectThat(err, Error(HasSubstr("OpenForReading")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *LoadTest) FileSystemSucceeds() {
	// File system
	f := &fakeFile{r: bytes.NewBufferString("taco")}
	ExpectCall(t.fs, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(f, nil))

	// Call
	blob, err := t.store.Load(blob.ComputeScore([]byte{}))
	AssertEq(nil, err)

	ExpectThat(blob, DeepEquals([]byte("taco")))
	ExpectTrue(f.closed)
}

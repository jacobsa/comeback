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
	"github.com/jacobsa/comeback/fs/mock"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/kv/disk"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"io"
	"path"
	"testing"
	"testing/iotest"
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
	store    kv.Store
}

func (t *diskStoreTest) SetUp(i *TestInfo) {
	var err error

	t.basePath = "/foo/bar"
	t.fs = mock_fs.NewMockFileSystem(i.MockController, "fs")

	t.store, err = disk.NewDiskKvStore(t.basePath, t.fs)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Set
////////////////////////////////////////////////////////////////////////

type SetTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&SetTest{}) }

func (t *SetTest) CallsFileSystem() {
	key := []byte("taco")
	val := []byte("burrito")

	// File system
	expectedPath := path.Join(t.basePath, "taco")
	ExpectCall(t.fs, "WriteFile")(expectedPath, DeepEquals(val), 0600).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.store.Set(key, val)
}

func (t *SetTest) FileSystemReturnsError() {
	// File system
	ExpectCall(t.fs, "WriteFile")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	err := t.store.Set([]byte{}, []byte{})

	ExpectThat(err, Error(HasSubstr("WriteFile")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *SetTest) FileSystemSaysOkay() {
	// File system
	ExpectCall(t.fs, "WriteFile")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	err := t.store.Set([]byte{}, []byte{})

	ExpectEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// Get
////////////////////////////////////////////////////////////////////////

type GetTest struct {
	diskStoreTest
}

func init() { RegisterTestSuite(&GetTest{}) }

func (t *GetTest) CallsFileSystem() {
	key := []byte("taco")

	// File system
	expectedPath := path.Join(t.basePath, "taco")
	ExpectCall(t.fs, "OpenForReading")(expectedPath).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.store.Get(key)
}

func (t *GetTest) FileSystemReturnsError() {
	// File system
	ExpectCall(t.fs, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	_, err := t.store.Get([]byte{})

	ExpectThat(err, Error(HasSubstr("OpenForReading")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *GetTest) ReadReturnsError() {
	// File system
	data := make([]byte, 1024)
	f := &fakeFile{r: iotest.TimeoutReader(bytes.NewBuffer(data))}
	ExpectCall(t.fs, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(f, nil))

	// Call
	_, err := t.store.Get([]byte{})

	ExpectThat(err, Error(HasSubstr("ReadAll")))
	ExpectThat(err, Error(HasSubstr("timeout")))
	ExpectTrue(f.closed)
}

func (t *GetTest) ReadSucceeds() {
	// File system
	f := &fakeFile{r: bytes.NewBufferString("taco")}
	ExpectCall(t.fs, "OpenForReading")(Any()).
		WillOnce(oglemock.Return(f, nil))

	// Call
	val, err := t.store.Get([]byte{})
	AssertEq(nil, err)

	ExpectThat(val, DeepEquals([]byte("taco")))
	ExpectTrue(f.closed)
}

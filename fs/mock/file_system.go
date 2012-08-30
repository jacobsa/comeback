// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_fs

import (
	fmt "fmt"
	fs "github.com/jacobsa/comeback/fs"
	io "io"
	oglemock "github.com/jacobsa/oglemock"
	runtime "runtime"
	unsafe "unsafe"
)

type MockFileSystem interface {
	fs.FileSystem
	oglemock.MockObject
}

type mockFileSystem struct {
	controller	oglemock.Controller
	description	string
}

func NewMockFileSystem(
	c oglemock.Controller,
	desc string) MockFileSystem {
	return &mockFileSystem{
		controller:	c,
		description:	desc,
	}
}

func (m *mockFileSystem) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockFileSystem) Oglemock_Description() string {
	return m.description
}

func (m *mockFileSystem) OpenForReading(p0 string) (o0 io.ReadCloser, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"OpenForReading",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockFileSystem.OpenForReading: invalid return values: %v", retVals))
	}

	// o0 io.ReadCloser
	if retVals[0] != nil {
		o0 = retVals[0].(io.ReadCloser)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockFileSystem) ReadDir(p0 string) (o0 []*fs.DirectoryEntry, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"ReadDir",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockFileSystem.ReadDir: invalid return values: %v", retVals))
	}

	// o0 []*fs.DirectoryEntry
	if retVals[0] != nil {
		o0 = retVals[0].([]*fs.DirectoryEntry)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

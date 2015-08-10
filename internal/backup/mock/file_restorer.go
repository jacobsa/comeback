// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_backup

import (
	fmt "fmt"
	os "os"
	runtime "runtime"
	unsafe "unsafe"

	backup "github.com/jacobsa/comeback/internal/backup"
	blob "github.com/jacobsa/comeback/internal/blob"
	oglemock "github.com/jacobsa/oglemock"
)

type MockFileRestorer interface {
	backup.FileRestorer
	oglemock.MockObject
}

type mockFileRestorer struct {
	controller  oglemock.Controller
	description string
}

func NewMockFileRestorer(
	c oglemock.Controller,
	desc string) MockFileRestorer {
	return &mockFileRestorer{
		controller:  c,
		description: desc,
	}
}

func (m *mockFileRestorer) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockFileRestorer) Oglemock_Description() string {
	return m.description
}

func (m *mockFileRestorer) RestoreFile(p0 []blob.Score, p1 string, p2 os.FileMode) (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"RestoreFile",
		file,
		line,
		[]interface{}{p0, p1, p2})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockFileRestorer.RestoreFile: invalid return values: %v", retVals))
	}

	// o0 error
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}

	return
}

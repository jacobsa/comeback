// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_backup

import (
	fmt "fmt"
	regexp "regexp"
	runtime "runtime"
	unsafe "unsafe"

	backup "github.com/jacobsa/comeback/internal/backup"
	blob "github.com/jacobsa/comeback/internal/blob"
	oglemock "github.com/jacobsa/oglemock"
)

type MockDirectorySaver interface {
	backup.DirectorySaver
	oglemock.MockObject
}

type mockDirectorySaver struct {
	controller  oglemock.Controller
	description string
}

func NewMockDirectorySaver(
	c oglemock.Controller,
	desc string) MockDirectorySaver {
	return &mockDirectorySaver{
		controller:  c,
		description: desc,
	}
}

func (m *mockDirectorySaver) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockDirectorySaver) Oglemock_Description() string {
	return m.description
}

func (m *mockDirectorySaver) Flush() (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Flush",
		file,
		line,
		[]interface{}{})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockDirectorySaver.Flush: invalid return values: %v", retVals))
	}

	// o0 error
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}

	return
}

func (m *mockDirectorySaver) Save(p0 string, p1 string, p2 []*regexp.Regexp) (o0 blob.Score, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Save",
		file,
		line,
		[]interface{}{p0, p1, p2})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockDirectorySaver.Save: invalid return values: %v", retVals))
	}

	// o0 blob.Score
	if retVals[0] != nil {
		o0 = retVals[0].(blob.Score)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}
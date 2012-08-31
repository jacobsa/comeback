// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_backup

import (
	backup "github.com/jacobsa/comeback/backup"
	blob "github.com/jacobsa/comeback/blob"
	fmt "fmt"
	io "io"
	oglemock "github.com/jacobsa/oglemock"
	runtime "runtime"
	unsafe "unsafe"
)

type MockFileSaver interface {
	backup.FileSaver
	oglemock.MockObject
}

type mockFileSaver struct {
	controller	oglemock.Controller
	description	string
}

func NewMockFileSaver(
	c oglemock.Controller,
	desc string) MockFileSaver {
	return &mockFileSaver{
		controller:	c,
		description:	desc,
	}
}

func (m *mockFileSaver) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockFileSaver) Oglemock_Description() string {
	return m.description
}

func (m *mockFileSaver) Save(p0 io.Reader) (o0 []blob.Score, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Save",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockFileSaver.Save: invalid return values: %v", retVals))
	}

	// o0 []blob.Score
	if retVals[0] != nil {
		o0 = retVals[0].([]blob.Score)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

type MockDirectorySaver interface {
	backup.DirectorySaver
	oglemock.MockObject
}

type mockDirectorySaver struct {
	controller	oglemock.Controller
	description	string
}

func NewMockDirectorySaver(
	c oglemock.Controller,
	desc string) MockDirectorySaver {
	return &mockDirectorySaver{
		controller:	c,
		description:	desc,
	}
}

func (m *mockDirectorySaver) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockDirectorySaver) Oglemock_Description() string {
	return m.description
}

func (m *mockDirectorySaver) Save(p0 string, p1 string) (o0 blob.Score, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Save",
		file,
		line,
		[]interface{}{p0, p1})

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

type MockLinkResolver interface {
	backup.LinkResolver
	oglemock.MockObject
}

type mockLinkResolver struct {
	controller	oglemock.Controller
	description	string
}

func NewMockLinkResolver(
	c oglemock.Controller,
	desc string) MockLinkResolver {
	return &mockLinkResolver{
		controller:	c,
		description:	desc,
	}
}

func (m *mockLinkResolver) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockLinkResolver) Oglemock_Description() string {
	return m.description
}

func (m *mockLinkResolver) Register(p0 int32, p1 uint64, p2 string) (o0 *string) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Register",
		file,
		line,
		[]interface{}{p0, p1, p2})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockLinkResolver.Register: invalid return values: %v", retVals))
	}

	// o0 *string
	if retVals[0] != nil {
		o0 = retVals[0].(*string)
	}

	return
}

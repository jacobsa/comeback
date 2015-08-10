// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_backup

import (
	fmt "fmt"
	runtime "runtime"
	unsafe "unsafe"

	backup "github.com/jacobsa/comeback/internal/backup"
	oglemock "github.com/jacobsa/oglemock"
)

type MockLinkResolver interface {
	backup.LinkResolver
	oglemock.MockObject
}

type mockLinkResolver struct {
	controller  oglemock.Controller
	description string
}

func NewMockLinkResolver(
	c oglemock.Controller,
	desc string) MockLinkResolver {
	return &mockLinkResolver{
		controller:  c,
		description: desc,
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

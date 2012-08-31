// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_sys

import (
	fmt "fmt"
	oglemock "github.com/jacobsa/oglemock"
	runtime "runtime"
	sys "github.com/jacobsa/comeback/sys"
	unsafe "unsafe"
)

type MockUserRegistry interface {
	sys.UserRegistry
	oglemock.MockObject
}

type mockUserRegistry struct {
	controller	oglemock.Controller
	description	string
}

func NewMockUserRegistry(
	c oglemock.Controller,
	desc string) MockUserRegistry {
	return &mockUserRegistry{
		controller:	c,
		description:	desc,
	}
}

func (m *mockUserRegistry) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockUserRegistry) Oglemock_Description() string {
	return m.description
}

func (m *mockUserRegistry) FindById(p0 sys.UserId) (o0 string, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FindById",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockUserRegistry.FindById: invalid return values: %v", retVals))
	}

	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(string)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockUserRegistry) FindByName(p0 string) (o0 sys.UserId, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FindByName",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockUserRegistry.FindByName: invalid return values: %v", retVals))
	}

	// o0 sys.UserId
	if retVals[0] != nil {
		o0 = retVals[0].(sys.UserId)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

type MockGroupRegistry interface {
	sys.GroupRegistry
	oglemock.MockObject
}

type mockGroupRegistry struct {
	controller	oglemock.Controller
	description	string
}

func NewMockGroupRegistry(
	c oglemock.Controller,
	desc string) MockGroupRegistry {
	return &mockGroupRegistry{
		controller:	c,
		description:	desc,
	}
}

func (m *mockGroupRegistry) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockGroupRegistry) Oglemock_Description() string {
	return m.description
}

func (m *mockGroupRegistry) FindById(p0 sys.GroupId) (o0 string, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FindById",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockGroupRegistry.FindById: invalid return values: %v", retVals))
	}

	// o0 string
	if retVals[0] != nil {
		o0 = retVals[0].(string)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockGroupRegistry) FindByName(p0 string) (o0 sys.GroupId, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"FindByName",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockGroupRegistry.FindByName: invalid return values: %v", retVals))
	}

	// o0 sys.GroupId
	if retVals[0] != nil {
		o0 = retVals[0].(sys.GroupId)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_crypto

import (
	fmt "fmt"
	crypto "github.com/jacobsa/comeback/crypto"
	oglemock "github.com/jacobsa/oglemock"
	runtime "runtime"
	unsafe "unsafe"
)

type MockKeyDeriver interface {
	crypto.KeyDeriver
	oglemock.MockObject
}

type mockKeyDeriver struct {
	controller  oglemock.Controller
	description string
}

func NewMockKeyDeriver(
	c oglemock.Controller,
	desc string) MockKeyDeriver {
	return &mockKeyDeriver{
		controller:  c,
		description: desc,
	}
}

func (m *mockKeyDeriver) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockKeyDeriver) Oglemock_Description() string {
	return m.description
}

func (m *mockKeyDeriver) Derive(p0 string, p1 []uint8) (o0 []uint8) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Derive",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockKeyDeriver.Derive: invalid return values: %v", retVals))
	}

	// o0 []uint8
	if retVals[0] != nil {
		o0 = retVals[0].([]uint8)
	}

	return
}

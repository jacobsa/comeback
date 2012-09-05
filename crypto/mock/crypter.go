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

type MockCrypter interface {
	crypto.Crypter
	oglemock.MockObject
}

type mockCrypter struct {
	controller  oglemock.Controller
	description string
}

func NewMockCrypter(
	c oglemock.Controller,
	desc string) MockCrypter {
	return &mockCrypter{
		controller:  c,
		description: desc,
	}
}

func (m *mockCrypter) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockCrypter) Oglemock_Description() string {
	return m.description
}

func (m *mockCrypter) Decrypt(p0 []uint8) (o0 []uint8, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Decrypt",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockCrypter.Decrypt: invalid return values: %v", retVals))
	}

	// o0 []uint8
	if retVals[0] != nil {
		o0 = retVals[0].([]uint8)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

func (m *mockCrypter) Encrypt(p0 []uint8) (o0 []uint8, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Encrypt",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockCrypter.Encrypt: invalid return values: %v", retVals))
	}

	// o0 []uint8
	if retVals[0] != nil {
		o0 = retVals[0].([]uint8)
	}

	// o1 error
	if retVals[1] != nil {
		o1 = retVals[1].(error)
	}

	return
}

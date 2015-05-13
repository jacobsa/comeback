// This file was auto-generated using createmock. See the following page for
// more information:
//
//     https://github.com/jacobsa/oglemock
//

package mock_blob

import (
	fmt "fmt"
	blob "github.com/jacobsa/comeback/blob"
	oglemock "github.com/jacobsa/oglemock"
	context "golang.org/x/net/context"
	runtime "runtime"
	unsafe "unsafe"
)

type MockStore interface {
	blob.Store
	oglemock.MockObject
}

type mockStore struct {
	controller  oglemock.Controller
	description string
}

func NewMockStore(
	c oglemock.Controller,
	desc string) MockStore {
	return &mockStore{
		controller:  c,
		description: desc,
	}
}

func (m *mockStore) Oglemock_Id() uintptr {
	return uintptr(unsafe.Pointer(m))
}

func (m *mockStore) Oglemock_Description() string {
	return m.description
}

func (m *mockStore) Contains(p0 context.Context, p1 blob.Score) (o0 bool) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Contains",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockStore.Contains: invalid return values: %v", retVals))
	}

	// o0 bool
	if retVals[0] != nil {
		o0 = retVals[0].(bool)
	}

	return
}

func (m *mockStore) Flush(p0 context.Context) (o0 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Flush",
		file,
		line,
		[]interface{}{p0})

	if len(retVals) != 1 {
		panic(fmt.Sprintf("mockStore.Flush: invalid return values: %v", retVals))
	}

	// o0 error
	if retVals[0] != nil {
		o0 = retVals[0].(error)
	}

	return
}

func (m *mockStore) Load(p0 context.Context, p1 blob.Score) (o0 []uint8, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Load",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockStore.Load: invalid return values: %v", retVals))
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

func (m *mockStore) Store(p0 context.Context, p1 []uint8) (o0 blob.Score, o1 error) {
	// Get a file name and line number for the caller.
	_, file, line, _ := runtime.Caller(1)

	// Hand the call off to the controller, which does most of the work.
	retVals := m.controller.HandleMethodCall(
		m,
		"Store",
		file,
		line,
		[]interface{}{p0, p1})

	if len(retVals) != 2 {
		panic(fmt.Sprintf("mockStore.Store: invalid return values: %v", retVals))
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

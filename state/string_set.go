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

package state

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sync"
)

// A string set represents a monitonically growing set of strings. It is safe
// to call any of its methods concurrently.
type StringSet interface {
	Add(str string)
	Contains(str string) bool
}

// Create an empty set.
func NewStringSet() StringSet {
	return &stringSet{
		elems: make(map[string]bool),
	}
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type stringSet struct {
	mutex     sync.RWMutex
	elems map[string]bool // Protected by mutex
}

func (s *stringSet) Add(str string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.elems[str] = true
}

func (s *stringSet) Contains(str string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.elems[str]
	return ok
}

func (s *stringSet) getElems() (elems []string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for elem, _ := range s.elems {
		elems = append(elems, elem)
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Gob encoding
////////////////////////////////////////////////////////////////////////

func init() {
	// Make sure that stringSets can be encoded where StringSet interface
	// variables are expected.
	gob.Register(&stringSet{})
}

func (s *stringSet) GobDecode(b []byte) (err error) {
	// Decode the list of elements.
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	var elems []string
	if err = decoder.Decode(&elems); err != nil {
		err = fmt.Errorf("Decoding list: %v", err)
		return
	}

	// Overwrite our map.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.elems = make(map[string]bool)
	for _, elem := range elems {
		s.elems[elem] = true
	}

	return
}

func (s *stringSet) GobEncode() (b []byte, err error) {
	// Get a list of elements, correctly dealing with locking.
	elems := s.getElems()

	// Encode the list.
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	if err = encoder.Encode(elems); err != nil {
		err = fmt.Errorf("Encoding list: %v", err)
		return
	}

	b = buf.Bytes()
	return
}

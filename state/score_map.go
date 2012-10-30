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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/sys"
	"os"
	"sync"
	"time"
)

// A map from stat info for a file to a set of scores that represented that
// file's contents at the time the stat info was collected. (This of course is
// not atomic, so it's more like "around the time that the stat info was
// collected".)
//
// All methods are safe for concurrent calling.
type ScoreMap interface {
	// Set a list of scores for a particular key.
	Set(key ScoreMapKey, scores []blob.Score)

	// Get the list of scores previously set for a key, or nil if no list has
	// been set.
	Get(key ScoreMapKey) (scores []blob.Score)
}

// Create an empty map.
func NewScoreMap() ScoreMap {
	return &scoreMap{
		elems: make(map[ScoreMapKey][]blob.Score),
	}
}

// Contains fields used by git for a similar purpose according to racy-git.txt.
type ScoreMapKey struct {
	Path        string
	Permissions os.FileMode
	Uid         sys.UserId
	Gid         sys.GroupId
	MTime       time.Time
	Inode       uint64
	Size        uint64
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type scoreMap struct {
	mutex sync.RWMutex
	elems map[ScoreMapKey][]blob.Score // Protected by mutex
}

func (s *scoreMap) Set(key ScoreMapKey, scores []blob.Score) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.elems[key] = scores
}

func (s *scoreMap) Get(key ScoreMapKey) (scores []blob.Score) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	scores, _ = s.elems[key]
	return
}

////////////////////////////////////////////////////////////////////////
// Gob encoding
////////////////////////////////////////////////////////////////////////

func init() {
	// Make sure that scoreMaps can be encoded where ScoreMap interface variables
	// are expected.
	gob.Register(&scoreMap{})
}

func (s *scoreMap) GobDecode(b []byte) (err error) {
	// Decode the map of elements.
	buf := bytes.NewBuffer(b)
	decoder := gob.NewDecoder(buf)

	elems := map[ScoreMapKey][]blob.Score{}
	if err = decoder.Decode(&elems); err != nil {
		err = fmt.Errorf("Decoding map: %v", err)
		return
	}

	// Overwrite our map.
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.elems = elems

	return
}

func (s *scoreMap) GobEncode() (b []byte, err error) {
	// Encode our map.
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if err = encoder.Encode(s.elems); err != nil {
		err = fmt.Errorf("Encoding map: %v", err)
		return
	}

	b = buf.Bytes()
	return
}

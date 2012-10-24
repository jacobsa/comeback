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
	"github.com/jacobsa/comeback/blob"
	"sync"
)

// A score set represents a monitonically growing set of blob scores. It is
// safe to call any of its methods concurrently. The zero value represents the
// empty set.
type ScoreSet struct {
	mutex sync.RWMutex
	hexScores map[string]bool  // Protected by mutex
}

func (s *ScoreSet) Add(score blob.Score) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.hexScores[score.Hex()] = true
}

func (s *ScoreSet) Contains(score blob.Score) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	_, ok := s.hexScores[score.Hex()]
	return ok
}

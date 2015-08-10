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
	"encoding/gob"
	"io"
	"time"

	"github.com/jacobsa/comeback/internal/util"
)

// State that should be saved between runs of the program.
type State struct {
	// The set of scores that are known to already exist in the blob store, in
	// hex form. It is unnecessary to again store any blob whose score is in this
	// set.
	ExistingScores util.StringSet

	// The time at which ExistingScores was last updated from the authoritative
	// source.
	RelistTime time.Time

	// A map from file system info to the scores that were seen for a given file
	// last time. These scores may have been written to the blob store, but not
	// flushed.
	ScoresForFiles ScoreMap
}

func LoadState(r io.Reader) (state State, err error) {
	decoder := gob.NewDecoder(r)
	err = decoder.Decode(&state)
	return
}

func SaveState(w io.Writer, state State) (err error) {
	encoder := gob.NewEncoder(w)
	err = encoder.Encode(state)
	return
}

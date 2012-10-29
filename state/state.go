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
)

// State that should be saved between runs of the program.
type State struct {
	// The set of scores that are known to already exist in the blob store,
	// represented in hex. It is unnecessary to again store any blob with one of
	// these scores.
	ExistingScores StringSet

	// A version number for the above set of scores. Any time state is saved,
	// this version should first be updated to a random number and saved to the
	// backup registry, making sure that the old version was still the current
	// one. This protects us from drifting out of date if another process is
	// concurrently adding scores to the blob store.
	ExistingScoresVersion uint64

	// A map from file system info to the scores that were seen for a given file
	// last time.
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

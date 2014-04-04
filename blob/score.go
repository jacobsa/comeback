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

package blob

import (
	"crypto/sha1"
	"fmt"
)

// A Score is the identifier for a blob previously stored by a blob store. It
// consists of a 20-byte SHA-1 hash of the blob's contents, so that with high
// probability two blobs have the same contents if and only if they have the
// same score.
const ScoreLength = 20

type Score [ScoreLength]byte

// Compute the score for the supplied blob. This is primarily intended for use
// by blob store implementations; most users should obtain scores through calls
// to a blob store's Store method.
func ComputeScore(b []byte) (s Score) {
	h := sha1.New()
	h.Write(b)

	slice := h.Sum(nil)
	if len(slice) != ScoreLength {
		panic(
			fmt.Sprintf(
				"Expected %d bytes for SHA-1; got %d",
				ScoreLength,
				len(slice)))
	}

	copy(s[:], slice)
	return
}

// Return a fixed-width hex version of the score's hash, suitable for using
// e.g. as a filename.
func (s Score) Hex() string {
	return fmt.Sprintf("%x", s)
}

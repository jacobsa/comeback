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

// Package blob contains types related to storage of content-addressed blobs.
package blob

import (
	"crypto/sha1"
	"fmt"
)

// A Score is the identifier for a blob previously stored by a blob store. It
// consists of a hash of the blob's contents, so that with high probability two
// blobs have the same contents if and only if they have the same score.
type Score interface {
	// Return the 20-byte 'raw' SHA-1 hash of the blob's contents.
	Sha1Hash() []byte
}

type score struct {
	hash []byte
}

func (s *score) Sha1Hash() []byte {
	return s.hash
}

// Compute the score for the supplied blob. This is primarily intended for use
// by blob store implementations; users should obtain only scores through calls
// to a store's Store method.
func ComputeScore(b []byte) Score {
	h := sha1.New()
	h.Write(b)
	return &score{hash: h.Sum(nil)}
}

// Return a fixed-width hex version of the score's hash, suitable for using
// e.g. as a filename.
func HexScore(score Score) string {
	return fmt.Sprintf("%x", score.Sha1Hash())
}

// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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
	"errors"

	"github.com/jacobsa/comeback/util"
)

// Create a blob store that wraps another, responding to calls as follows:
//
//  *  Flush will panic; it is assumed that any buffering and early returning
//     is in front of this store, and that the wrapped store responds to Store
//     only when the blob is durable.
//
//  *  Contains will be responded to directly by this store based on the
//     contents of existingScores. It is assumed that existingScores initially
//     contains only scores that are durable in the wrapped store.
//
//  *  Store will be forwarded to the wrapped store. When it succeeds,
//     existingScores will be updated.
//
//  *  Load will be forwarded to the wrapped store.
//
func NewExistingScoresStore(
	existingScores util.StringSet,
	wrapped Store) (store Store) {
	store = &existingScoresStore{
		scores:  existingScores,
		wrapped: wrapped,
	}

	return
}

type existingScoresStore struct {
	scores  util.StringSet
	wrapped Store
}

func (bs *existingScoresStore) Store(blob []byte) (s Score, err error) {
	err = errors.New("TODO")
	return
}

func (bs *existingScoresStore) Flush() (err error) {
	panic("We expect buffering to happen outside of here")
}

func (bs *existingScoresStore) Contains(score Score) (b bool) {
	b = bs.scores.Contains(score.Hex())
	return
}

func (bs *existingScoresStore) Load(s Score) (blob []byte, err error) {
	blob, err = bs.wrapped.Load(s)
	return
}

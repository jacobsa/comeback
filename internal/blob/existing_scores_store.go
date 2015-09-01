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
	"github.com/jacobsa/comeback/internal/util"
	"golang.org/x/net/context"
)

// Create a blob store that wraps another, responding immediately to calls to
// Store for content that already exists in the wrapped blob store. For calls
// that are passed on, this store will fill in StoreRequest.score.
//
// existingScores must initially be a subset of the scores contained by the
// wrapped store, in hex form. It will be updated upon successful calls to
// wrapped.Store.
func Internal_NewExistingScoresStore(
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

func (bs *existingScoresStore) Store(
	ctx context.Context,
	req *StoreRequest) (s Score, err error) {
	s = ComputeScore(req.Blob)

	// Do we need to do anything?
	if bs.scores.Contains(s.Hex()) {
		return
	}

	// Pass on the blob to the wrapped store, saving it the trouble of
	// recomputing the score. If it is successful, remember that fact.
	req.score = s
	_, err = bs.wrapped.Store(ctx, req)
	if err == nil {
		bs.scores.Add(s.Hex())
	}

	return
}

func (bs *existingScoresStore) Load(
	ctx context.Context,
	s Score) (blob []byte, err error) {
	blob, err = bs.wrapped.Load(ctx, s)
	return
}

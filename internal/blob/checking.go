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
	"fmt"

	"golang.org/x/net/context"
)

// Return a blob store that wraps the supplied one, confirming that the blob
// contents and scores it returns are correct, guarding against silent data
// corruption.
func NewCheckingStore(wrapped Store) Store {
	return &checkingStore{wrapped}
}

type checkingStore struct {
	wrapped Store
}

func (s *checkingStore) Store(
	ctx context.Context,
	blob []byte) (score Score, err error) {
	// Call the wrapped store.
	if score, err = s.wrapped.Store(ctx, blob); err != nil {
		return
	}

	// Check its result.
	expected := ComputeScore(blob)
	if score != expected {
		err = fmt.Errorf(
			"Incorrect score returned for blob; %s vs %s.",
			score.Hex(),
			expected.Hex())

		return
	}

	return
}

func (s *checkingStore) Load(
	ctx context.Context,
	score Score) (blob []byte, err error) {
	// Call the wrapped store.
	if blob, err = s.wrapped.Load(ctx, score); err != nil {
		return
	}

	// Check its result.
	actual := ComputeScore(blob)
	if actual != score {
		return nil, fmt.Errorf(
			"Incorrect data returned for blob; requested score is %s actual is %s.",
			score.Hex(),
			actual.Hex())
	}

	return
}

func (s *checkingStore) Contains(ctx context.Context, score Score) (b bool) {
	b = s.wrapped.Contains(ctx, score)
	return
}

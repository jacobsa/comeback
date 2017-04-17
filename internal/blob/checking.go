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
	"context"
	"fmt"
)

// Return a blob store that wraps the supplied one, confirming that the blob
// contents it loads are correct, guarding against silent data corruption.
func Internal_NewCheckingStore(wrapped Store) Store {
	return &checkingStore{wrapped}
}

type checkingStore struct {
	wrapped Store
}

func (s *checkingStore) Save(
	ctx context.Context,
	req *SaveRequest) (score Score, err error) {
	return s.wrapped.Save(ctx, req)
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

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

import "context"

// A Store knows how to save blobs for later retrieval.
type Store interface {
	// Store a blob, returning a score with which it can later be retrieved.
	Save(
		ctx context.Context,
		req *SaveRequest) (s Score, err error)

	// Load a previously-stored blob.
	Load(ctx context.Context, s Score) (blob []byte, err error)
}

type SaveRequest struct {
	// The blob data to be stored.
	Blob []byte

	// The score of the blob, used in a conspiracy between existingScoresStore
	// and downstream stores.
	score Score

	// A buffer for holding the result of encryption. If the user reuses the
	// request struct, we can reuse this buffer.
	ciphertext []byte
}

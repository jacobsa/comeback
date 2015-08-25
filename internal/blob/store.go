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

import "golang.org/x/net/context"

// A Store knows how to store blobs for later retrieval.
type Store interface {
	// Store a blob, returning a score with which it can later be retrieved.
	Store(
		ctx context.Context,
		req *StoreRequest) (s Score, err error)

	// Return true only if the supplied score is durable in the blob store.
	// Implementations may choose to return false if the information is not
	// available.
	Contains(ctx context.Context, score Score) (b bool)

	// Load a previously-stored blob.
	Load(ctx context.Context, s Score) (blob []byte, err error)
}

type StoreRequest struct {
	// The blob data to be stored.
	blob []byte
}

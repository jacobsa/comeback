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

// A Store knows how to store blobs for later retrieval.
type Store interface {
	// Store the supplied blob, returning a score with which it can later be
	// retrieved. Note that the blob is not guaranteed to be durable until Flush
	// is successfully called.
	Store(blob []byte) (s Score, err error)

	// Flush previous stored blobs to durable storage. Store must not be called
	// again.
	Flush() (err error)

	// Return true only if the supplied score is in the blob store and will be
	// durable by the time of a successful Flush call.
	Contains(score Score) (b bool)

	// Load a previously-stored blob.
	Load(s Score) (blob []byte, err error)
}

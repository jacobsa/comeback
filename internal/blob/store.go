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
	"github.com/jacobsa/comeback/internal/crypto"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

// A Store knows how to store blobs for later retrieval.
type Store interface {
	// Store a blob, returning a score with which it can later be retrieved.
	Store(
		ctx context.Context,
		req *StoreRequest) (s Score, err error)

	// Load a previously-stored blob.
	Load(ctx context.Context, s Score) (blob []byte, err error)
}

type StoreRequest struct {
	// The blob data to be stored.
	Blob []byte

	// The score of the blob, used in a conspiracy between existingScoresStore
	// and downstream stores.
	score Score
}

// Create a blob store that stores blobs in the supplied bucket under the given
// name prefix, encrypting with the supplied crypter.
//
// existingScores must contain only scores that are known to exist in the
// bucket, in hex form. It will be updated as the blob store is used.
func NewStore(
	bucket gcs.Bucket,
	objectNamePrefix string,
	crypter crypto.Crypter,
	existingScores util.StringSet) (bs Store, err error) {
	// Store blobs in GCS.
	bs = Internal_NewGCSStore(bucket, objectNamePrefix)

	// Don't make redundant calls to GCS.
	bs = Internal_NewExistingScoresStore(existingScores, bs)

	// Make paranoid checks on the results.
	bs = Internal_NewCheckingStore(bs)

	// Encrypt blob data before sending it off to GCS.
	bs = Internal_NewEncryptingStore(crypter, bs)

	return
}

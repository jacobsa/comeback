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

package wiring

import (
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/util"
	"github.com/jacobsa/gcloud/gcs"
)

const (
	BlobObjectNamePrefix = "blobs/"

	// A generous lower bound for where the OS starts to tell us to fuck off if
	// we have too many files. This may also cover the case where we get "no such
	// host" errors, apparently because we do too many lookups all at once.
	osFileLimit = 96
)

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// Create a blob store that stores blobs in the supplied bucket, encrypting
// with the supplied crypter.
//
// existingScores must contain only scores that are known to exist in the
// bucket, in hex form. It will be updated as the blob store is used.
func MakeBlobStore(
	bucket gcs.Bucket,
	crypter crypto.Crypter,
	existingScores util.StringSet) (bs blob.Store, err error) {
	// Store blobs in GCS.
	bs = blob.NewGCSStore(bucket, BlobObjectNamePrefix)

	// Respond efficiently to Contains requests.
	bs = blob.NewExistingScoresStore(existingScores, bs)

	// Buffer around GCS with bounded parallelism, allowing file system scanning
	// to proceed independent of waiting for GCS to ack writes.
	const latencySecs = 2
	const bandwidthBytesPerSec = 50e6
	const bandwidthHz = 512

	bs = blob.NewBufferingStore(
		3*bandwidthBytesPerSec*latencySecs,
		minInt(osFileLimit, 3*bandwidthHz*latencySecs),
		bs)

	// Make paranoid checks on the results.
	bs = blob.NewCheckingStore(bs)

	// Encrypt blob data before sending it off to GCS.
	bs = blob.NewEncryptingStore(crypter, bs)

	return
}

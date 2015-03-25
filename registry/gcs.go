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

package registry

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a registry that stores data in the supplied GCS bucket, deriving a
// crypto key from the supplied password and ensuring that the bucket may not
// in the future be used with any other key and has not in the past, either.
// Return a crypter configured to use the key.
func NewGCSRegistry(
	bucket gcs.Bucket,
	cryptoPassword string,
	deriver crypto.KeyDeriver) (r Registry, crypter crypto.Crypter, err error) {
	return newGCSRegistry(
		bucket,
		cryptoPassword,
		deriver,
		crypto.NewCrypter,
		rand.Reader)
}

const (
	gcsJobKeyPrefix      = "jobs/"
	gcsMetadataKey_Name  = "job_name"
	gcsMetadataKey_Score = "hex_score"
)

// A registry that stores job records in a GCS bucket. Object names are of the
// form
//
//     <gcsJobKeyPrefix><time>
//
// where <time> is a time.Time with UTC location formatted according to
// time.RFC3339Nano. Additional information is stored as object metadata fields
// keyed by the constants above. Metadata fields are used in preference to
// object content so that they are accessible on a ListObjects request.
//
// The bucket additionally contains a "marker" object (named by the constant
// markerItemName) with metadata keys specifying a salt and a ciphertext for
// some random plaintext, generated ant written the first time the bucket is
// used. This marker allows us to verify that the user-provided crypto password
// is correct by deriving a key using the password and the salt and making sure
// that the ciphertext can be decrypted using that key.
type gcsRegistry struct {
	bucket gcs.Bucket
}

// Like NewGCSRegistry, but with more injected.
func newGCSRegistry(
	bucket gcs.Bucket,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func(key []byte) (crypto.Crypter, error),
	cryptoRandSrc io.Reader) (r Registry, crypter crypto.Crypter, err error) {
}

func (r *gcsRegistry) RecordBackup(j CompletedJob) (err error) {
	err = fmt.Errorf("gcsRegistry.RecordBackup is not implemented.")
	return
}

func (r *gcsRegistry) ListRecentBackups() (jobs []CompletedJob, err error) {
	err = fmt.Errorf("gcsRegistry.ListRecentBackups is not implemented.")
	return
}

func (r *gcsRegistry) FindBackup(
	startTime time.Time) (job CompletedJob, err error) {
	err = fmt.Errorf("gcsRegistry.FindBackup is not implemented.")
	return
}

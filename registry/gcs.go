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
	deriver crypto.KeyDeriver,
) (r Registry, crypter crypto.Crypter, err error)

const (
	gcsMetadataKey_Name  = "job_name"
	gcsMetadataKey_Score = "hex_score"
)

// A registry that stores job records in a GCS bucket. Object names are of the
// form
//
//     <gcsKeyPrefix><time>
//
// where <time> is a time.Time with UTC location formatted according to
// time.RFC3339Nano. Additional information is stored as object metadata fields
// keyed by the constants above. Metadata fields are used in preference to
// object content so that they are accessible on a ListObjects request.
type gcsRegistry struct {
}

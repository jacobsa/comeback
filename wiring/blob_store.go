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

// Create a blob store that stores blobs in the supplied bucket, encrypting
// with the supplied crypter.
//
// existingScores must contain only scores that are known to exist in the
// bucket, in hex form. It will be updated as the blob store is used.
func makeBlobStore(
	bucket gcs.Bucket,
	crypter crypto.Crypter,
	existingScores util.StringSet) (s blob.Store, err error)

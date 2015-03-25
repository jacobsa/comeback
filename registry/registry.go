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

package registry

import (
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/gcloud/gcs"
)

type Registry interface {
	// Record that the named backup job has completed.
	RecordBackup(j CompletedJob) (err error)

	// Return a list of the most recent completed backups.
	ListRecentBackups() (jobs []CompletedJob, err error)

	// Find a particular completed job by ID.
	FindBackup(jobId uint64) (job CompletedJob, err error)
}

// Create a registry that stores data in the supplied GCS bucket, deriving a
// crypto key from the supplied password and ensuring that the bucket may not
// in the future be used with any other key and has not in the past, either.
// Return a crypter configured to use the key.
func NewRegistry(
	bucket gcs.Bucket,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
) (r Registry, crypter crypto.Crypter, err error)

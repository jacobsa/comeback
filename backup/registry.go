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

package backup

import (
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"time"
)

// A record in the backup registry describing a successful backup job.
type CompletedJob struct {
	// The name of the backup job.
	Name string

	// The time at which the backup was started.
	StartTime time.Time

	// The score representing the contents of the backup.
	Score blob.Score
}

type Registry interface {
	// Record that the named backup job has completed.
	RecordBackup(j CompletedJob) (err error)

	// Return a list of the most recent completed backups.
	ListRecentBackups() (jobs []CompletedJob, err error)
}

// Create a registry that stores data in the supplied SimpleDB domain.
//
// Before doing so, check to see whether this domain has been used as a
// registry before. If not, write an encrypted marker with the supplied
// crypter. If it has been used before, make sure that it was used with a
// crypter compatible with the supplied one. This prevents accidentally writing
// data with the wrong key, as if the user entered the wrong password.
//
// The crypter must be set up such that it is guaranteed to return an error if
// it is used to decrypt ciphertext encrypted with a different key. In that
// case, this function will return an *IncompatibleCrypterError.
func NewRegistry(crypter crypto.Crypter, domain sdb.Domain) (r Registry, err error) {
	err = fmt.Errorf("TODO")
	return
}

type IncompatibleCrypterError struct {
}

func (e *IncompatibleCrypterError) Error() string {
	return "TODO"
}

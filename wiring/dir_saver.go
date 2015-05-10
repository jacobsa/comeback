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
	"fmt"

	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/state"
	"github.com/jacobsa/comeback/util"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a directory saver that stores blobs in the supplied bucket,
// encrypting with a key derived from the given password. If the bucket has
// been used by comeback in the past, the password must match the password used
// previously or an error will be returned.
//
// existingScores must contain only scores that are known to exist in the blob
// store, in hex form. It will be updated as the directory saver is used.
//
// scoresForFiles is a cache from file system info to the scores that were seen
// at the time that file was stat'd, to be used in saving the work of reading
// file contents each time. It will also be updated by the directory saver.
//
// TODO(jacobsa): Make sure to test the password error behavior. See issue #20.
// TODO(jacobsa): Make sure to test the existingScores behavior. See issue #20.
// TODO(jacobsa): Make sure to test the scoresForFiles behavior. See issue #20.
func MakeDirSaver(
	password string,
	bucket gcs.Bucket,
	existingScores util.StringSet,
	scoresForFiles state.ScoreMap) (ds backup.DirectorySaver, err error) {
	// Create a crypter from the supplied password, verifying it against any past
	// use of the bucket.
	_, crypter, err := makeRegistryAndCrypter(password, bucket)
	if err != nil {
		err = fmt.Errorf("makeRegistryAndCrypter: %v", err)
		return
	}

	// Wrap a blob store around the bucket. Tell it to encrypt using the crypter.
	bs, err := makeBlobStore(bucket, crypter, existingScores)
	if err != nil {
		err = fmt.Errorf("makeBlobStore: %v", err)
		return
	}

	// Create a file saver that writes to the blob store.
	fileSaver, err := backup.NewFileSaver(bs, chunkSize, fs)
	if err != nil {
		err = fmt.Errorf("NewFileSaver: %v", err)
		return
	}

	// Create a directory saver that shares the blob store with the file saver.
	ds, err := backup.NewDirectorySaver(bs, fs, fileSaver)
	if err != nil {
		err = fmt.Errorf("NewDirectorySaver: %v", err)
		return
	}

	return
}

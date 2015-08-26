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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/backup"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a directory restorer that reads from teh supplied bucket, decrypting
// with a key derived from the given password.
func MakeDirRestorer(
	ctx context.Context,
	password string,
	bucket gcs.Bucket) (dr backup.DirectoryRestorer, err error) {
	// Use the real file system.
	fs, err := makeFileSystem()
	if err != nil {
		err = fmt.Errorf("makeFileSystem: %v", err)
		return
	}

	// Create a crypter from the supplied password, verifying it against any past
	// use of the bucket.
	_, crypter, err := MakeRegistryAndCrypter(ctx, password, bucket)
	if err != nil {
		err = fmt.Errorf("MakeRegistryAndCrypter: %v", err)
		return
	}

	// Wrap a blob store around the bucket. Tell it to decrypt using the crypter.
	bs, err := MakeBlobStore(bucket, crypter, util.NewStringSet())
	if err != nil {
		err = fmt.Errorf("MakeBlobStore: %v", err)
		return
	}

	// Create a file restorer that reads from the blob store.
	fr, err := makeFileRestorer(bs, fs)
	if err != nil {
		err = fmt.Errorf("makeFileRestorer: %v", err)
		return
	}

	// Create a directory restorer that shares the blob store with the file
	// restorer.
	dr, err = backup.NewDirectoryRestorer(bs, fs, fr)
	if err != nil {
		err = fmt.Errorf("NewDirectoryRestorer: %v", err)
		return
	}

	return
}

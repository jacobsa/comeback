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
	"crypto/sha256"
	"fmt"

	"github.com/jacobsa/comeback/internal/crypto"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a registry for the supplied bucket, given the supplied crypto key
// password.
func MakeRegistryAndCrypter(
	password string,
	bucket gcs.Bucket) (r registry.Registry, crypter crypto.Crypter, err error) {
	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32 // Minimum key length for AES-SIV
	keyDeriver := crypto.NewPbkdf2KeyDeriver(pbkdf2Iters, keyLen, sha256.New)

	// Create the registry and crypter.
	r, crypter, err = registry.NewGCSRegistry(bucket, password, keyDeriver)
	if err != nil {
		err = fmt.Errorf("NewGCSRegistry: %v", err)
		return
	}

	return
}

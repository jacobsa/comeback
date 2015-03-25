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
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/jacobsa/aws/sdb"
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

// Like NewGCSRegistry, but with more injected.
func newGCSRegistry(
	bucket gcs.Bucket,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func(key []byte) (crypto.Crypter, error),
	cryptoRandSrc io.Reader) (r Registry, crypter crypto.Crypter, err error) {
	// Ask for the previously-written encrypted marker and password salt, if any.
	attrs, err := domain.GetAttributes(
		markerItemName,
		false, // No need to ask for a consistent read
		[]string{encryptedDataMarker, passwordSaltMarker},
	)

	if err != nil {
		err = fmt.Errorf("GetAttributes: %v", err)
		return
	}

	// If we got back any attributes, we must verify that they are compatible.
	if len(attrs) > 0 {
		crypter, err = verifyCompatibleAndSetUpCrypter(
			attrs,
			cryptoPassword,
			deriver,
			createCrypter,
		)

		if err != nil {
			return
		}

		// All is good.
		r = &registry{
			crypter,
			domain.Db(),
			domain,
			runInRetryLoop,
		}

		return
	}

	// Otherwise, we want to claim this domain. Encrypt some random data, base64
	// encode it, then write it out. Make sure to use a precondition to defeat
	// the race condition where another machine is doing the same simultaneously.

	// Generate a random salt.
	salt := make([]byte, 8)
	if _, err = io.ReadAtLeast(cryptoRandSrc, salt, len(salt)); err != nil {
		err = fmt.Errorf("Reading random bytes for salt: %v", err)
		return
	}

	// Derive a crypto key and create the crypter.
	cryptoKey := deriver.DeriveKey(cryptoPassword, salt)
	crypter, err = createCrypter(cryptoKey)
	if err != nil {
		err = fmt.Errorf("createCrypter: %v", err)
		return
	}

	// Create some plaintext.
	plaintext := make([]byte, 8)
	if _, err = io.ReadAtLeast(cryptoRandSrc, plaintext, len(plaintext)); err != nil {
		err = fmt.Errorf("Reading random bytes for plaintext: %v", err)
		return
	}

	// Encrypt the plaintext.
	ciphertext, err := crypter.Encrypt(plaintext)
	if err != nil {
		err = fmt.Errorf("Encrypt: %v", err)
		return
	}

	// SimpleDB requires only UTF-8 text.
	encodedEncryptedData := base64.StdEncoding.EncodeToString(ciphertext)
	encodedSalt := base64.StdEncoding.EncodeToString(salt)

	// Write out the two markers.
	err = domain.PutAttributes(
		markerItemName,
		[]sdb.PutUpdate{
			sdb.PutUpdate{Name: encryptedDataMarker, Value: encodedEncryptedData},
			sdb.PutUpdate{Name: passwordSaltMarker, Value: encodedSalt},
		},
		&sdb.Precondition{Name: encryptedDataMarker, Value: nil},
	)

	if err != nil {
		err = fmt.Errorf("PutAttributes: %v", err)
		return
	}

	// All is good.
	r = &registry{
		crypter,
		domain.Db(),
		domain,
		runInRetryLoop,
	}

	return
}

func verifyCompatibleAndSetUpCrypter(
	ciphertextBase64 string,
	passwordSaltBase64 string,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func(key []byte) (crypto.Crypter, error)) (
	crypter crypto.Crypter,
	err error) {
	// Base64-decode.
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		err = fmt.Errorf("Decoding ciphertext: %v", err)
		return
	}

	passwordSalt, err := base64.StdEncoding.DecodeString(passwordSaltBase64)
	if err != nil {
		err = fmt.Errorf("Decoding password salt: %v", err)
		return
	}

	// Derive a key and create a crypter.
	cryptoKey := deriver.DeriveKey(cryptoPassword, passwordSalt)
	if crypter, err = createCrypter(cryptoKey); err != nil {
		err = fmt.Errorf("createCrypter: %v", err)
		return
	}

	// Attempt to decrypt the ciphertext.
	if _, err = crypter.Decrypt(ciphertext); err != nil {
		// Special case: Did the crypter signal that the key was wrong?
		if _, ok := err.(*crypto.NotAuthenticError); ok {
			err = fmt.Errorf("The supplied password is incorrect.")
			return
		}

		// Generic error.
		err = fmt.Errorf("Decrypt: %v", err)
		return
	}

	return
}

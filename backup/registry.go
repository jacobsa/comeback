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
	"encoding/base64"
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"math/rand"
	"time"
)

const (
	// The item name we use for "domain is already in use" markers.
	markerItemName = "comeback_marker"

	// The attribute name we use for storing crypto-compatibility data.
	cryptoMarkerAttributeName = "encrypted_data"
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

func verifyCompatible(
	markerAttr sdb.Attribute,
	crypter crypto.Crypter) (err error) {
	// The ciphertext should be base64-encoded.
	encoded := markerAttr.Value
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		err = fmt.Errorf("base64.DecodeString(%s): %v", encoded, err)
		return
	}

	// Attempt to decrypt the ciphertext.
	if _, err = crypter.Decrypt(ciphertext); err != nil {
		// Special case: Did the crypter signal that the key was wrong?
		if _, ok := err.(*crypto.NotAuthenticError); ok {
			err = &IncompatibleCrypterError{
				"The supplied crypter is not compatible with the data in the domain.",
			}
			return
		}

		// Generic error.
		err = fmt.Errorf("Decrypt: %v", err)
		return
	}

	return
}

// Create a registry that stores data in the named SimpleDB domain.
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
func NewRegistry(
	crypter crypto.Crypter,
	db sdb.SimpleDB,
	domainName string) (r Registry, err error) {
	// Attempt to open the domain.
	domain, err := db.OpenDomain(domainName)
	if err != nil {
		err = fmt.Errorf("OpenDomain: %v", err)
		return
	}

	// Set up a tentative result.
	r = &registry{crypter, db, domain}

	// Ask for the data that will tell us whether the crypter is compatible with
	// the previous one used in this domain, if any.
	attrs, err := domain.GetAttributes(
		markerItemName,
		false,  // No need to ask for a consistent read
		[]string{cryptoMarkerAttributeName},
	)

	if err != nil {
		err = fmt.Errorf("GetAttributes: %v", err)
		return
	}

	// If we got back an attribute, we must verify that it is compatible.
	if len(attrs) > 0 {
		err = verifyCompatible(attrs[0], crypter)
		return
	}

	// Otherwise, we want to claim this domain. Encrypt some random data, base64
	// encode it, then write it out. Make sure to use a precondition to defeat
	// the race condition where another machine is doing the same simultaneously.
	plaintext := getRandBytes()
	ciphertext, err := crypter.Encrypt(plaintext)
	if err != nil {
		err = fmt.Errorf("Encrypt: %v", err)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	err = domain.PutAttributes(
		markerItemName,
		[]sdb.PutUpdate{
			sdb.PutUpdate{Name: cryptoMarkerAttributeName, Value: encoded},
		},
		&sdb.Precondition{Name: cryptoMarkerAttributeName, Value: nil},
	)

	if err != nil {
		err = fmt.Errorf("PutAttributes: %v", err)
		return
	}

	// All is good.
	return
}

func getRandBytes() []byte {
	a := rand.Uint32()
	b := rand.Uint32()

	return []byte{
		byte(a),
		byte(a >> 8),
		byte(a >> 16),
		byte(a >> 24),
		byte(b),
		byte(b >> 8),
		byte(b >> 16),
		byte(b >> 24),
	}
}

type registry struct {
	crypter crypto.Crypter
	db sdb.SimpleDB
	domain sdb.Domain
}

func (r *registry) RecordBackup(j CompletedJob) (err error) {
	err = fmt.Errorf("TODO")
	return
}

func (r *registry) ListRecentBackups() (jobs []CompletedJob, err error) {
	err = fmt.Errorf("TODO")
	return
}

type IncompatibleCrypterError struct {
	s string
}

func (e *IncompatibleCrypterError) Error() string {
	return e.s
}

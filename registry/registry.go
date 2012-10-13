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
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"io"
	"math/rand"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	// The item name we use for "domain is already in use" markers.
	markerItemName = "comeback_marker"

	// The attribute names we use for storing crypto-compatibility data.
	encryptedDataMarker = "encrypted_data"
	passwordSaltMarker = "password_salt"

	// A time format that works properly with range queries.
	iso8601TimeFormat = "2006-01-02T15:04:05Z"
)

// A regexp for selected item names.
var itemNameRegexp = regexp.MustCompile(`^backup_([0-9a-f]{16})$`)

type Registry interface {
	// Record that the named backup job has completed.
	RecordBackup(j CompletedJob) (err error)

	// Return a list of the most recent completed backups.
	ListRecentBackups() (jobs []CompletedJob, err error)

	// Find a particular completed job by ID.
	FindBackup(jobId uint64) (job CompletedJob, err error)
}

// Create a registry that stores data in the supplied SimpleDB domain, deriving
// a crypto key from the supplied password and ensuring that the domain may not
// in the future be used with any other key and has not in the past, either.
// Return a crypter configured to use the key.
func NewRegistry(
	domain sdb.Domain,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
) (r Registry, crypter crypto.Crypter, err error) {
	return newRegistry(
		domain,
		cryptoPassword,
		deriver,
		crypto.NewCrypter,
		crypto_rand.Reader)
}

func verifyCompatibleAndSetUpCrypter(
	markerAttrs []sdb.Attribute,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func(key []byte) (crypto.Crypter, error),
) (crypter crypto.Crypter, err error) {
	// Look through the attributes for what we need.
	var ciphertext []byte
	var salt []byte

	for _, attr := range markerAttrs {
		var dest *[]byte
		switch attr.Name {
		case encryptedDataMarker:
			dest = &ciphertext
		case passwordSaltMarker:
			dest = &salt
		default:
			continue
		}

		// The data is base64-encoded.
		if *dest, err = base64.StdEncoding.DecodeString(attr.Value); err != nil {
			err = fmt.Errorf("Decoding %s (%s): %v", attr.Name, attr.Value, err)
			return
		}
	}

	// Did we get both ciphertext and salt?
	if ciphertext == nil {
		err = fmt.Errorf("Missing encrypted data marker.")
		return
	}

	if salt == nil {
		err = fmt.Errorf("Missing password salt marker.")
		return
	}

	// Derive a key and create a crypter.
	cryptoKey := deriver.DeriveKey(cryptoPassword, salt)
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

// A version split out for testability.
func newRegistry(
	domain sdb.Domain,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func(key []byte) (crypto.Crypter, error),
	cryptoRandSrc io.Reader,
) (r Registry, crypter crypto.Crypter, err error) {
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
		r = &registry{crypter, domain.Db(), domain}
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
	r = &registry{crypter, domain.Db(), domain}

	return
}

func get8RandBytes(src *rand.Rand) []byte {
	a := src.Uint32()
	b := src.Uint32()

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
	db      sdb.SimpleDB
	domain  sdb.Domain
}

func (r *registry) RecordBackup(job CompletedJob) (err error) {
	// Job names must be valid SimpleDB attribute values.
	if len(job.Name) == 0 || len(job.Name) > 1024 || !utf8.ValidString(job.Name) {
		err = fmt.Errorf(
			"Job names must be non-empty UTF-8 no more than 1024 bytes long.")
		return
	}

	// Format the item name.
	itemName := sdb.ItemName(fmt.Sprintf("backup_%016x", job.Id))

	// Use a time format that works correctly with range queries. Make sure to
	// standardize on UTC.
	formattedTime := job.StartTime.UTC().Format(iso8601TimeFormat)

	// Call the domain.
	updates := []sdb.PutUpdate{
		sdb.PutUpdate{Name: "job_name", Value: job.Name},
		sdb.PutUpdate{Name: "start_time", Value: formattedTime},
		sdb.PutUpdate{Name: "score", Value: job.Score.Hex()},
	}

	precond := sdb.Precondition{Name: "score", Value: nil}

	if err = r.domain.PutAttributes(itemName, updates, &precond); err != nil {
		err = fmt.Errorf("PutAttributes: %v", err)
		return
	}

	return
}

func convertAttributes(attrs []sdb.Attribute) (j CompletedJob, err error) {
	for _, attr := range attrs {
		switch attr.Name {
		case "job_name":
			j.Name = attr.Value

		case "start_time":
			if j.StartTime, err = time.Parse(iso8601TimeFormat, attr.Value); err != nil {
				err = fmt.Errorf("Invalid start_time value: %v", err)
				return
			}

		case "score":
			var decoded []byte
			decoded, err = hex.DecodeString(attr.Value)
			if err != nil || len(decoded) != 20 {
				err = fmt.Errorf("Invalid score: %s", attr.Value)
				return
			}

			j.Score = blob.Score(decoded)
		}
	}

	// Everything must have been returned.
	if j.Name == "" {
		err = fmt.Errorf("Missing job_name attribute.")
		return
	}

	if j.StartTime.IsZero() {
		err = fmt.Errorf("Missing start_time attribute.")
		return
	}

	if j.Score == nil {
		err = fmt.Errorf("Missing score attribute.")
		return
	}

	return
}

func convertSelectedItem(item sdb.SelectedItem) (j CompletedJob, err error) {
	// Convert the item's attributes.
	if j, err = convertAttributes(item.Attributes); err != nil {
		return
	}

	// Convert the item name.
	subMatches := itemNameRegexp.FindStringSubmatch(string(item.Name))
	if subMatches == nil {
		err = fmt.Errorf("Invalid item name: %s", item.Name)
		return
	}

	if j.Id, err = strconv.ParseUint(subMatches[1], 16, 64); err != nil {
		panic(fmt.Sprintf("Unexpected result for name: %s", item.Name))
	}

	return
}

func (r *registry) ListRecentBackups() (jobs []CompletedJob, err error) {
	// Call the database.
	query := fmt.Sprintf(
		"select job_name, start_time, score from `%s` where "+
			"start_time is not null order by start_time desc",
		r.domain.Name(),
	)

	results, _, err := r.db.Select(
		query,
		false, // No need for consistent reads.
		nil,
	)

	if err != nil {
		err = fmt.Errorf("Select: %v", err)
		return
	}

	// Convert each result.
	for _, item := range results {
		var job CompletedJob
		job, err = convertSelectedItem(item)
		if err != nil {
			err = fmt.Errorf("Item %s is invalid: %v", item.Name, err)
			return
		}

		jobs = append(jobs, job)
	}

	return
}

func (r *registry) FindBackup(jobId uint64) (job CompletedJob, err error) {
	// Call the domain.
	attrs, err := r.domain.GetAttributes(
		sdb.ItemName(fmt.Sprintf("backup_%016x", jobId)),
		false, // Consistent read unnecessary
		[]string{"job_name", "start_time", "score"},
	)

	if err != nil {
		err = fmt.Errorf("GetAttributes: %v", err)
		return
	}

	// Convert the results.
	if job, err = convertAttributes(attrs); err != nil {
		err = fmt.Errorf("Returned attributes invalid: %v", err)
		return
	}

	job.Id = jobId
	return
}

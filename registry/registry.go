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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"math/rand"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	// The item name we use for "domain is already in use" markers.
	markerItemName = "comeback_marker"

	// The attribute name we use for storing crypto-compatibility data.
	cryptoMarkerAttributeName = "encrypted_data"

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
			err = fmt.Errorf("The supplied password is incorrect.")
			return
		}

		// Generic error.
		err = fmt.Errorf("Decrypt: %v", err)
		return
	}

	return
}

// Create a registry that stores data in the supplied SimpleDB domain,
// encrypting using a key derived from the supplied password.
//
// Before doing so, check to see whether this domain has been used as a
// registry before. If not, write an encrypted marker. If it has been used
// before, make sure that it was used with the same password. This prevents
// accidentally writing data with the wrong key when the user enters the wrong
// password.
//
// The crypter must be set up such that it is guaranteed to return an error if
// it is used to decrypt ciphertext encrypted with a different key.
func NewRegistry(
	domain sdb.Domain,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	randSrc *rand.Rand,
) (r Registry, err error) {
	return newRegistry(
		domain,
		cryptoPassword,
		deriver,
		crypto.NewCrypter,
		randSrc)
}

// A version split out for testability.
func newRegistry(
	domain sdb.Domain,
	cryptoPassword string,
	deriver crypto.KeyDeriver,
	createCrypter func (key []byte) (crypto.Crypter, error),
	randSrc *rand.Rand,
) (r Registry, err error) {
	// Set up a tentative result.
	r = &registry{crypter, domain.Db(), domain}

	// Ask for the data that will tell us whether the crypter is compatible with
	// the previous one used in this domain, if any.
	attrs, err := domain.GetAttributes(
		markerItemName,
		false, // No need to ask for a consistent read
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
	plaintext := get8RandBytes(randSrc)
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

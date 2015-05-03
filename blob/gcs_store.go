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

package blob

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
)

// A key placed in GCS object metadata by GCSStore containing the hex SHA-1
// expected for the object contents. This is of course redundant with the
// object name; we use it as a paranoid check against GCS returning the
// metadata or contents for the wrong object.
const GCSMetadataKey_SHA1Hex = "comeback_sha1_hex"

// A key placed in GCS object metadata by GCSStore containing the hex MD5 sum
// expected for the object contents. If GCS reports a different MD5 sum or
// returns contents with a different MD5 sum, we know something screwy has
// happened.
//
// See here for more info: https://github.com/jacobsa/comeback/issues/18
const GCSMetadataKey_MD5Hex = "comeback_md5_hex"

// Return a blob store that stores blobs in the supplied GCS bucket. GCS object
// names look like:
//
//     <prefix><score>
//
// where <score> is the result of calling Score.Hex.
//
// The blob store trusts that it has full ownership of this portion of the
// bucket's namespace -- if a score name exists, then it points to the correct
// data.
//
// The returned store does not support Flush or Contains; these methods must
// not be called.
func NewGCSStore(
	bucket gcs.Bucket,
	prefix string) (store *GCSStore) {
	store = &GCSStore{
		bucket:     bucket,
		namePrefix: prefix,
	}

	return
}

type GCSStore struct {
	bucket     gcs.Bucket
	namePrefix string
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *GCSStore) makeName(score Score) (name string) {
	name = s.namePrefix + score.Hex()
	return
}

// Verify the internal consistency of the object record, and return the score
// of the blob that it represents.
func (s *GCSStore) parseObject(o *gcs.Object) (score Score, err error) {
	// Is the name of the appropriate form?
	if !strings.HasPrefix(o.Name, s.namePrefix) {
		err = fmt.Errorf("Unexpected object name: %q", o.Name)
		return
	}

	// Parse the hex score.
	hexScore := strings.TrimPrefix(o.Name, s.namePrefix)
	score, err = ParseHexScore(hexScore)
	if err != nil {
		err = fmt.Errorf("Unexpected hex score %q: %v", hexScore, err)
		return
	}

	// We expect the hex score to match the hex SHA-1 in the metadata.
	hexSHA1, ok := o.Metadata[GCSMetadataKey_SHA1Hex]
	if !ok {
		err = fmt.Errorf(
			"Object %q is missing metadata key %q",
			o.Name,
			GCSMetadataKey_SHA1Hex)
		return
	}

	if hexSHA1 != hexScore {
		err = fmt.Errorf(
			"Score/SHA-1 metadata mismatch for object %q: %q",
			o.Name,
			hexSHA1)
		return
	}

	// We expect the hex MD5 in the object metadata to align with what GCS says
	// the object's MD5 is.
	hexMD5, ok := o.Metadata[GCSMetadataKey_MD5Hex]
	if !ok {
		err = fmt.Errorf(
			"Object %q is missing metadata key %q",
			o.Name,
			GCSMetadataKey_MD5Hex)
		return
	}

	if hex.DecodedLen(len(hexMD5)) != md5.Size {
		err = fmt.Errorf(
			"Object %q has weird hex MD5 metadata: %q",
			o.Name,
			hexMD5)
		return
	}

	var md5 [md5.Size]byte
	_, err = hex.Decode(md5[:], []byte(hexMD5))
	if err != nil {
		err = fmt.Errorf(
			"Object %q has invalid hex MD5 in metadata: %q",
			o.Name,
			hexMD5)
		return
	}

	if md5 != o.MD5 {
		err = fmt.Errorf(
			"MD5 mismatch for object %q: %s vs. %s",
			o.Name,
			hex.EncodeToString(md5[:]),
			hex.EncodeToString(o.MD5[:]))
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *GCSStore) Store(blob []byte) (score Score, err error) {
	// Compute a score and an object name.
	score = ComputeScore(blob)
	name := s.makeName(score)

	// Create the object.
	crc32c := *gcsutil.CRC32C(blob)
	md5 := *gcsutil.MD5(blob)
	sha1 := sha1.Sum(blob)

	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: bytes.NewReader(blob),
		CRC32C:   &crc32c,
		MD5:      &md5,

		Metadata: map[string]string{
			GCSMetadataKey_SHA1Hex: hex.EncodeToString(sha1[:]),
			GCSMetadataKey_MD5Hex:  hex.EncodeToString(md5[:]),
		},
	}

	o, err := s.bucket.CreateObject(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	// Paranoid check: what we get back from GCS should match what we put in.
	if o.CRC32C != crc32c {
		panic(fmt.Sprintf(
			"CRC32C mismatch for object %q: 0x%08xv vs. 0x%08x",
			o.Name,
			o.CRC32C,
			crc32c))
	}

	if o.MD5 != md5 {
		panic(fmt.Sprintf(
			"MD5 mismatch for object %q: %s vs. %s",
			o.Name,
			hex.EncodeToString(o.MD5[:]),
			hex.EncodeToString(md5[:])))
	}

	return
}

func (s *GCSStore) Flush() (err error) {
	panic("GCSStore.Flush not supported; wiring code bug?")
}

func (s *GCSStore) Contains(score Score) (b bool) {
	panic("GCSStore.Contains not supported; wiring code bug?")
}

func (s *GCSStore) Load(score Score) (blob []byte, err error) {
	// Create a ReadCloser.
	req := &gcs.ReadObjectRequest{
		Name: s.makeName(score),
	}

	rc, err := s.bucket.NewReader(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	// Read from it.
	blob, err = ioutil.ReadAll(rc)
	if err != nil {
		rc.Close()
		err = fmt.Errorf("ReadAll: %v", err)
		return
	}

	// Close it.
	err = rc.Close()
	if err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	return
}

// List all of the blobs that are known to be durable in the bucket.
func (s *GCSStore) List() (scores []Score, err error) {
	req := &gcs.ListObjectsRequest{
		Prefix: s.namePrefix,
	}

	// List repeatedly until we're done.
	for {
		// Call the bucket.
		var listing *gcs.Listing
		listing, err = s.bucket.ListObjects(context.Background(), req)
		if err != nil {
			err = fmt.Errorf("ListObjects: %v", err)
			return
		}

		// Process results.
		for _, o := range listing.Objects {
			var score Score
			score, err = s.parseObject(o)
			if err != nil {
				return
			}

			scores = append(scores, score)
		}

		// Continue?
		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	return
}

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
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
)

// A key placed in GCS object metadata by gcsStore containing the hex SHA-1
// expected for the object contents. This is of course redundant with the
// object name; we use it as a paranoid check against GCS returning the
// metadata or contents for the wrong object.
const metadataKey_SHA1 = "comeback_sha1"

// A key placed in GCS object metadata by gcsStore containing the CRC32C
// checksum expected for the object contents. If GCS reports a different
// checksum or returns contents with a different checksum, we know something
// screwy has happened.
//
// See here for more info: https://github.com/jacobsa/comeback/issues/18
const metadataKey_CRC32C = "comeback_crc32c"

// A key placed in GCS object metadata by gcsStore containing the hex MD5 sum
// expected for the object contents. If GCS reports a different MD5 sum or
// returns contents with a different MD5 sum, we know something screwy has
// happened.
//
// See here for more info: https://github.com/jacobsa/comeback/issues/18
const metadataKey_MD5 = "comeback_md5"

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
// Awkward interface: the resulting store requires SaveRequest.score fields to
// be filled in by the caller. This is accomplished by ensuring that it is
// wrapped by a store created with NewEncryptingStore.
func NewGCSStore(
	bucket gcs.Bucket,
	prefix string) (store Store) {
	store = &gcsStore{
		bucket:     bucket,
		namePrefix: prefix,
	}

	return
}

type gcsStore struct {
	bucket     gcs.Bucket
	namePrefix string
}

var _ Store = &gcsStore{}

// Parse and verify the internal consistency of the supplied object record in
// the same manner that a gcsStore configured with the supplied object name
// prefix would. Return the score of the blob that the object contains.
func ParseObjectRecord(
	o *gcs.Object,
	namePrefix string) (score Score, err error) {
	// Is the name of the appropriate form?
	if !strings.HasPrefix(o.Name, namePrefix) {
		err = fmt.Errorf("Unexpected object name: %q", o.Name)
		return
	}

	// Parse the hex score.
	hexScore := strings.TrimPrefix(o.Name, namePrefix)
	score, err = ParseHexScore(hexScore)
	if err != nil {
		err = fmt.Errorf(
			"Unexpected hex score %q for object %q: %v",
			hexScore,
			o.Name,
			err)
		return
	}

	// We expect the hex score to match the hex SHA-1 in the metadata.
	hexSHA1, ok := o.Metadata[metadataKey_SHA1]
	if !ok {
		err = fmt.Errorf(
			"Object %q is missing metadata key %q",
			o.Name,
			metadataKey_SHA1)
		return
	}

	if hexSHA1 != hexScore {
		err = fmt.Errorf(
			"Score/SHA-1 metadata mismatch for object %q: %q",
			o.Name,
			hexSHA1)
		return
	}

	// We expect the hex CRC32C in the object metadata to match what GCS says the
	// object's checksum is.
	hexCRC32C, ok := o.Metadata[metadataKey_CRC32C]
	if !ok {
		err = fmt.Errorf(
			"Object %q is missing metadata key %q",
			o.Name,
			metadataKey_CRC32C)
		return
	}

	crc32Uint64, err := strconv.ParseUint(hexCRC32C, 0, 32)
	if err != nil {
		err = fmt.Errorf(
			"Object %q has invalid hex CRC32C %q: %v",
			o.Name,
			hexCRC32C,
			err)
		return
	}

	if uint32(crc32Uint64) != o.CRC32C {
		err = fmt.Errorf(
			"CRC32C mismatch for object %q: %#08x vs. %#08x",
			o.Name,
			crc32Uint64,
			o.CRC32C)
		return
	}

	// We expect the hex MD5 in the object metadata to match what GCS says the
	// object's MD5 sum is.
	hexMD5, ok := o.Metadata[metadataKey_MD5]
	if !ok {
		err = fmt.Errorf(
			"Object %q is missing metadata key %q",
			o.Name,
			metadataKey_MD5)
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

	if o.MD5 == nil {
		err = fmt.Errorf("MD5 missing for object %q", o.Name)
		return
	}

	if md5 != *o.MD5 {
		err = fmt.Errorf(
			"MD5 mismatch for object %q: %s vs. %s",
			o.Name,
			hex.EncodeToString(md5[:]),
			hex.EncodeToString(o.MD5[:]))
		return
	}

	return
}

// Write object records for all of the blob objects in the supplied bucket into
// the given channel, without closing it. The order of records is undefined.
// The caller will likely want to call ParseObjectRecord for each record.
func ListBlobObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	namePrefix string,
	objects chan<- *gcs.Object) (err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// GCS object listing is slow. Parallelize sixteen ways.
	const hexDigits = "0123456789abcdef"
	for i := 0; i < len(hexDigits); i++ {
		digit := hexDigits[i]
		eg.Go(func() (err error) {
			err = gcsutil.ListPrefix(ctx, bucket, namePrefix+string(digit), objects)
			return
		})
	}

	err = eg.Wait()
	return
}

// Feed the output of ListBlobObjects into ParseObjectRecord, passing on the
// scores to the supplied channel without closing it.
func ListScores(
	ctx context.Context,
	bucket gcs.Bucket,
	namePrefix string,
	scores chan<- Score) (err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// List object records into a channel.
	objects := make(chan *gcs.Object, 100)
	eg.Go(func() (err error) {
		defer close(objects)
		err = ListBlobObjects(ctx, bucket, namePrefix, objects)
		if err != nil {
			err = fmt.Errorf("ListBlobObjects: %v", err)
			return
		}

		return
	})

	// Parse and verify records, and write out scores.
	eg.Go(func() (err error) {
		for o := range objects {
			// Parse and verify.
			var score Score
			score, err = ParseObjectRecord(o, namePrefix)
			if err != nil {
				err = fmt.Errorf("ParseObjectRecord: %v", err)
				return
			}

			// Send on the score.
			select {
			case scores <- score:

			// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		return
	})

	err = eg.Wait()
	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

func (s *gcsStore) makeName(score Score) (name string) {
	name = s.namePrefix + score.Hex()
	return
}

////////////////////////////////////////////////////////////////////////
// Public interface
////////////////////////////////////////////////////////////////////////

func (s *gcsStore) Save(
	ctx context.Context,
	req *SaveRequest) (score Score, err error) {
	blob := req.Blob

	// Pull out the score and choose an object name.
	score = req.score
	name := s.makeName(score)

	// Optimization: we know that the score is the SHA-1 hash of the blob, so
	// don't need to compute it again.
	var sha1 []byte = score[:]

	// Create the object.
	crc32c := *gcsutil.CRC32C(blob)
	md5 := *gcsutil.MD5(blob)

	createReq := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: bytes.NewReader(blob),
		CRC32C:   &crc32c,
		MD5:      &md5,

		Metadata: map[string]string{
			metadataKey_SHA1:   hex.EncodeToString(sha1),
			metadataKey_CRC32C: fmt.Sprintf("%#08x", crc32c),
			metadataKey_MD5:    hex.EncodeToString(md5[:]),
		},
	}

	o, err := s.bucket.CreateObject(ctx, createReq)
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

	if o.MD5 == nil {
		err = fmt.Errorf("MD5 missing for object %q", o.Name)
	}

	if *o.MD5 != md5 {
		panic(fmt.Sprintf(
			"MD5 mismatch for object %q: %s vs. %s",
			o.Name,
			hex.EncodeToString(o.MD5[:]),
			hex.EncodeToString(md5[:])))
	}

	return
}

func (s *gcsStore) Load(
	ctx context.Context,
	score Score) (blob []byte, err error) {
	// Create a ReadCloser.
	req := &gcs.ReadObjectRequest{
		Name: s.makeName(score),
	}

	rc, err := s.bucket.NewReader(ctx, req)
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

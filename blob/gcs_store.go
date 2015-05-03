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
	"fmt"
	"io/ioutil"
	"strings"

	"golang.org/x/net/context"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
)

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
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: bytes.NewReader(blob),
		CRC32C:   gcsutil.CRC32C(blob),
		MD5:      gcsutil.MD5(blob),
	}

	_, err = s.bucket.CreateObject(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
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

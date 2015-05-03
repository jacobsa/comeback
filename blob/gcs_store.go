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

func (s *gcsStore) Store(blob []byte) (score Score, err error) {
	// Compute a score and an object name.
	score = ComputeScore(blob)
	name := s.makeName(score)

	// Create the object.
	//
	// TODO(jacobsa): Set MD5 and CRC32C. See issue #18.
	req := &gcs.CreateObjectRequest{
		Name:     name,
		Contents: bytes.NewReader(blob),
	}

	_, err = s.bucket.CreateObject(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	return
}

func (s *gcsStore) Flush() (err error) {
	panic("gcsStore.Flush not supported; wiring code bug?")
}

func (s *gcsStore) Contains(score Score) (b bool) {
	panic("gcsStore.Contains not supported; wiring code bug?")
}

func (s *gcsStore) Load(score Score) (blob []byte, err error) {
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
func (s *gcsStore) List() (scores []Score, err error) {
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
			if !strings.HasPrefix(o.Name, s.namePrefix) {
				err = fmt.Errorf("Unexpected object name: %q", o.Name)
				return
			}

			var score Score
			hexScore := strings.TrimPrefix(o.Name, s.namePrefix)
			score, err = ParseHexScore(hexScore)
			if err != nil {
				err = fmt.Errorf("Unexpected hex score %q: %v", hexScore, err)
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

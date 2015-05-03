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
	prefix string) (store Store)

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

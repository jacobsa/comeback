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

// Package gcskv implements a key/value store in a Google Cloud Storage bucket.
package kv

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
	"google.golang.org/cloud/storage"
)

// Create a key/value store that stores data in the supplied GCS bucket. Keys
// supplied to its methods must be valid GCS object names. It is assumed that
// no keys in the bucket are ever removed.
//
// This function blocks while listing keys in the bucket.
func NewGCSStore(bucket gcs.Bucket) (Store, error) {
	store := &gcsStore{
		bucket: bucket,
	}

	return store, nil
}

type gcsStore struct {
	bucket gcs.Bucket
}

func (s *gcsStore) Set(key string, val []byte) (err error) {
	req := &gcs.CreateObjectRequest{
		Attrs: storage.ObjectAttrs{
			Name: key,
		},
		Contents: bytes.NewReader(val),
	}

	_, err = s.bucket.CreateObject(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	return
}

func (s *gcsStore) Get(key string) (val []byte, err error) {
	// Create a ReadCloser.
	req := &gcs.ReadObjectRequest{
		Name: key,
	}

	rc, err := s.bucket.NewReader(context.Background(), req)
	if err != nil {
		err = fmt.Errorf("NewReader: %v", err)
		return
	}

	// Read from it.
	val, err = ioutil.ReadAll(rc)
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

func (s *gcsStore) Contains(key string) (res bool, err error) {
	// Unsupported.
	res = false
	return
}

func (s *gcsStore) ListKeys(prefix string) (keys []string, err error) {
	q := &storage.Query{
		Prefix: prefix,
	}

	for q != nil {
		// Grab one set of results.
		var listing *storage.Objects

		listing, err = s.bucket.ListObjects(context.Background(), q)
		if err != nil {
			return
		}

		// Sanity check.
		if len(listing.Prefixes) != 0 {
			err = fmt.Errorf("Unexpected prefixes in listing.")
			return
		}

		// Accumulate the results.
		for _, o := range listing.Results {
			keys = append(keys, o.Name)
		}

		// Move on to the next query, if necessary.
		q = listing.Next
	}

	return
}

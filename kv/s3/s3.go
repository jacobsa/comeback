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

// Package s3 implements a key/value store in an Amazon S3 bucket.
package s3

import (
	"fmt"
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/comeback/kv"
)

// Create a key/value store that stores data in the supplied S3 bucket. Keys
// supplied to its methods must be valid S3 keys. It is assumed that no keys in
// the bucket are ever removed.
//
// This function blocks while listing keys in the bucket.
func NewS3KvStore(bucket s3.Bucket) (kv.Store, error) {
	store := &kvStore{
		bucket:    bucket,
	}

	return store, nil
}

func getAllKeys(bucket s3.Bucket) ([]string, error) {
	keys := []string{}
	for {
		var prevKey string
		if len(keys) > 0 {
			prevKey = keys[len(keys)-1]
		}

		partialKeys, err := bucket.ListKeys(prevKey)
		if err != nil {
			return nil, fmt.Errorf("ListKeys: %v", err)
		}

		if len(partialKeys) == 0 {
			break
		}

		keys = append(keys, partialKeys...)
	}

	return keys, nil
}

type kvStore struct {
	bucket s3.Bucket
}

func (s *kvStore) Set(key []byte, val []byte) error {
	// Call the bucket.
	if err := s.bucket.StoreObject(string(key), val); err != nil {
		return fmt.Errorf("StoreObject: %v", err)
	}

	return nil
}

func (s *kvStore) Get(key []byte) (val []byte, err error) {
	if val, err = s.bucket.GetObject(string(key)); err != nil {
		err = fmt.Errorf("GetObject: %v", err)
		return
	}

	return
}

func (s *kvStore) Contains(key []byte) (res bool, err error) {
	// Unsupported.
	res = false
	return
}

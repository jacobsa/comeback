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
	return nil, fmt.Errorf("TODO")
}
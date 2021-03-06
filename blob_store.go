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

package main

import (
	"context"
	"sync"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/wiring"
)

var gBlobStoreOnce sync.Once
var gBlobStore blob.Store

func makeBlobStore(ctx context.Context) (bs blob.Store, err error) {
	bucket := getBucket(ctx)
	crypter := getCrypter(ctx)
	state := getState(ctx)

	// Store blobs in GCS.
	bs = blob.NewGCSStore(bucket, wiring.BlobObjectNamePrefix)

	// Don't make redundant calls to GCS.
	bs = blob.NewExistingScoresStore(state.ExistingScores, bs)

	// Make paranoid checks on the results.
	bs = blob.NewCheckingStore(bs)

	// Encrypt blob data before sending it off to GCS.
	bs = blob.NewEncryptingStore(crypter, bs)

	return
}

func initBlobStore(ctx context.Context) {
	var err error

	gBlobStore, err = makeBlobStore(ctx)
	if err != nil {
		panic(err)
	}
}

func getBlobStore(ctx context.Context) blob.Store {
	gBlobStoreOnce.Do(func() { initBlobStore(ctx) })
	return gBlobStore
}

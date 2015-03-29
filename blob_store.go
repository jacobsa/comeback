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

package main

import (
	"sync"

	"github.com/jacobsa/comeback/blob"
)

const (
	blobKeyPrefix = "blobs/"
)

var g_blobStoreOnce sync.Once
var g_blobStore blob.Store

func initBlobStore() {
	kvStore := getKvStore()
	crypter := getCrypter()

	// Store blobs in a key/value store.
	const latencySecs = 2
	const bandwidthBytesPerSec = 20e6
	const bandwidthHz = 8

	g_blobStore = blob.NewKVStoreBlobStore(
		kvStore,
		blobKeyPrefix,
		3*bandwidthBytesPerSec*latencySecs,
		3*bandwidthHz*latencySecs)

	// Make sure the values returned by the key/value store aren't corrupted.
	g_blobStore = blob.NewCheckingStore(g_blobStore)

	// Encrypt blob data.
	g_blobStore = blob.NewEncryptingStore(crypter, g_blobStore)
}

func getBlobStore() blob.Store {
	g_blobStoreOnce.Do(initBlobStore)
	return g_blobStore
}

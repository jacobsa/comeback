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
	"log"
	"sync"
	"time"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/util"
)

const (
	blobObjectNamePrefix = "blobs/"
)

var g_blobStoreOnce sync.Once
var g_blobStore blob.Store

// A generous lower bound for where the OS starts to tell us to fuck off if we
// have too many files. This may also cover the case where we get "no such
// host" errors, apparently because we do too many lookups all at once.
const osFileLimit = 128

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func initBlobStore() {
	bucket := getBucket()
	crypter := getCrypter()
	stateStruct := getState()

	// Store blobs in GCS.
	g_blobStore = blob.NewGCSStore(bucket, blobObjectNamePrefix)

	// If we don't know the set of hex scores in the store, or the set of scores
	// is stale, re-list.
	age := time.Now().Sub(stateStruct.RelistTime)
	const maxAge = 30 * 24 * time.Hour

	if stateStruct.ExistingScores == nil || age > maxAge {
		log.Println("Listing existing scores...")

		stateStruct.RelistTime = time.Now()
		allScores, err := g_blobStore.List()
		if err != nil {
			log.Fatalln("g_blobStore.List:", err)
		}

		log.Printf(
			"Listed %d scores in %v.",
			len(allScores),
			time.Since(stateStruct.RelistTime))

		stateStruct.ExistingScores = util.NewStringSet()
		for _, score := range allScores {
			stateStruct.ExistingScores.Add(score.Hex())
		}
	}

	// Store blobs in a key/value store.
	const latencySecs = 2
	const bandwidthBytesPerSec = 50e6
	const bandwidthHz = 512

	g_blobStore = blob.NewKVStoreBlobStore(
		kvStore,
		blobKeyPrefix,
		3*bandwidthBytesPerSec*latencySecs,
		minInt(osFileLimit, 3*bandwidthHz*latencySecs))

	// Make sure the values returned by the key/value store aren't corrupted.
	g_blobStore = blob.NewCheckingStore(g_blobStore)

	// Encrypt blob data.
	g_blobStore = blob.NewEncryptingStore(crypter, g_blobStore)
}

func getBlobStore() blob.Store {
	g_blobStoreOnce.Do(initBlobStore)
	return g_blobStore
}

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
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/aws/s3/s3util"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/state"
	s3_kv "github.com/jacobsa/comeback/kv/s3"
	"log"
	"sync"
)

var g_kvStoreOnce sync.Once
var g_kvStore kv.Store

func initKvStore() {
	cfg := getConfig()
	var err error

	// Open a connection to S3.
	bucket, err := s3.OpenBucket(cfg.S3Bucket, cfg.S3Region, cfg.AccessKey)
	if err != nil {
		log.Fatalln("Creating S3 bucket:", err)
	}

	// If we don't know the set of keys in S3, find out.
	stateStruct := getState()
	if stateStruct.ExistingScores == nil {
		log.Println("Listing keys in S3 bucket...")
		allKeys, err := s3util.ListAllKeys(bucket)
		if err != nil {
			log.Fatalln("Creating S3 bucket:", err)
		}

		log.Println("Listed", len(allKeys), "keys.")

		stateStruct.ExistingScores = state.NewStringSet()
		for _, key := range allKeys {
			stateStruct.ExistingScores.Add(key)
		}
	}

	// Store keys and values in S3.
	g_kvStore, err = s3_kv.NewS3KvStore(bucket)
	if err != nil {
		log.Fatalln("Creating S3 kv store:", err)
	}

	// Respond efficiently to Contains requests.
	g_kvStore = kv.NewExistingKeysStore(stateStruct.ExistingScores, g_kvStore)
}

func getKvStore() kv.Store {
	g_kvStoreOnce.Do(initKvStore)
	return g_kvStore
}

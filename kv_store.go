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

	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/state"
)

func makeBasicKeyStore() kv.Store

var g_kvStoreOnce sync.Once
var g_kvStore kv.Store

func initKvStore() {
	// Create the underlying key store.
	g_kvStore = makeBasicKeyStore()

	// If we don't know the set of keys in the store, or the set of keys is
	// stale, re-list.
	stateStruct := getState()
	age := time.Now().Sub(stateStruct.RelistTime)
	const maxAge = 30 * 24 * time.Hour

	if stateStruct.ExistingKeys == nil || age > maxAge {
		log.Println("Listing existing keys...")

		stateStruct.RelistTime = time.Now()
		allKeys, err := g_kvStore.ListKeys("")
		if err != nil {
			log.Fatalln("g_kvStore.List:", err)
		}

		log.Println("Listed", len(allKeys), "keys.")

		stateStruct.ExistingKeys = state.NewStringSet()
		for _, key := range allKeys {
			stateStruct.ExistingKeys.Add(key)
		}
	}

	// Respond efficiently to Contains requests.
	g_kvStore = state.NewExistingKeysStore(stateStruct.ExistingKeys, g_kvStore)
}

func getKvStore() kv.Store {
	g_kvStoreOnce.Do(initKvStore)
	return g_kvStore
}

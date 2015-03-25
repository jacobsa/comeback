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
	"crypto/sha256"
	"log"
	"sync"

	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/registry"
	"github.com/jacobsa/util/password"
)

var g_registryAndCrypterOnce sync.Once
var g_registry registry.Registry
var g_crypter crypto.Crypter

func initRegistryAndCrypter() {
	var err error
	bucket := getBucket()

	// Grab config info.
	cfg := getConfig()

	// Read in the crypto password.
	cryptoPassword := password.ReadPassword("Entry crypto password: ")
	if len(cryptoPassword) == 0 {
		log.Fatalln("You must enter a password.")
	}

	// Derive a crypto key from the password using PBKDF2, recommended for use by
	// NIST Special Publication 800-132. The latter says that PBKDF2 is approved
	// for use with HMAC and any approved hash function. Special Publication
	// 800-107 lists SHA-256 as an approved hash function.
	const pbkdf2Iters = 4096
	const keyLen = 32 // Minimum key length for AES-SIV
	keyDeriver := crypto.NewPbkdf2KeyDeriver(pbkdf2Iters, keyLen, sha256.New)

	// Create the registry and crypter.
	g_registry, g_crypter, err = registry.NewGCSRegistry(
		bucket,
		cryptoPassword,
		keyDeriver)

	if err != nil {
		log.Fatalln("Creating registry:", err)
	}
}

func getRegistry() registry.Registry {
	g_registryAndCrypterOnce.Do(initRegistryAndCrypter)
	return g_registry
}

func getCrypter() crypto.Crypter {
	g_registryAndCrypterOnce.Do(initRegistryAndCrypter)
	return g_crypter
}

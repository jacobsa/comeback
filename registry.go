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
	"fmt"
	"log"
	"sync"

	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/registry"
	"github.com/jacobsa/comeback/wiring"
)

var gRegistryAndCrypterOnce sync.Once
var gRegistry registry.Registry
var gCrypter crypto.Crypter

func initRegistryAndCrypter() {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	bucket := getBucket()
	password := getPassword()

	gRegistry, gCrypter, err = wiring.MakeRegistryAndCrypter(
		password,
		bucket)

	if err != nil {
		err = fmt.Errorf("MakeRegistryAndCrypter: %v", err)
		return
	}
}

func getRegistry() registry.Registry {
	gRegistryAndCrypterOnce.Do(initRegistryAndCrypter)
	return gRegistry
}

func getCrypter() crypto.Crypter {
	gRegistryAndCrypterOnce.Do(initRegistryAndCrypter)
	return gCrypter
}

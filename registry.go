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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/crypto"
	"github.com/jacobsa/comeback/internal/registry"
	"github.com/jacobsa/comeback/internal/wiring"
)

var gRegistryAndCrypterOnce sync.Once
var gRegistry registry.Registry
var gCrypter crypto.Crypter

func initRegistryAndCrypter(ctx context.Context) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	bucket := getBucket(ctx)
	password := getPassword()

	gRegistry, gCrypter, err = wiring.MakeRegistryAndCrypter(
		password,
		bucket)

	if err != nil {
		err = fmt.Errorf("MakeRegistryAndCrypter: %v", err)
		return
	}
}

func getRegistry(ctx context.Context) registry.Registry {
	gRegistryAndCrypterOnce.Do(func() { initRegistryAndCrypter(ctx) })
	return gRegistry
}

func getCrypter(ctx context.Context) crypto.Crypter {
	gRegistryAndCrypterOnce.Do(func() { initRegistryAndCrypter(ctx) })
	return gCrypter
}

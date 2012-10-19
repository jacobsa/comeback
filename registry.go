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
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/registry"
	"log"
	"sync"
)

var g_registryOnce sync.Once
var g_registry registry.Registry
var g_crypter crypto.Crypter

func getRegistryAndCrypter() (registry.Registry, crypto.Crypter) {
	g_registryOnce.Do(func() {
		log.Fatalln("TODO: g_registryOnce")
	})

	return g_registry, g_crypter
}

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

package crypto

import (
	"hash"
)

// An object that knows how to derive a crypto key, given a user password and a
// random salt.
type KeyDeriver interface {
	Derive(password string, salt []byte) []byte
}

// Create a key deriver that uses the PBKDF2 key derivation function of
// RFC 2898 / PKCS #5 v2.0.
func NewPbkdf2KeyDeriver(iters int, keyLen int, h func() hash.Hash) KeyDeriver

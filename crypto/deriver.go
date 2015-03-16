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

	"golang.org/x/crypto/pbkdf2"
)

// An object that knows how to derive a crypto key, given a user password and a
// random salt.
type KeyDeriver interface {
	DeriveKey(password string, salt []byte) []byte
}

// Create a key deriver that uses the PBKDF2 key derivation function of
// RFC 2898 / PKCS #5 v2.0.
func NewPbkdf2KeyDeriver(iters int, keyLen int, h func() hash.Hash) KeyDeriver {
	return &pbkdf2KeyDeriver{iters, keyLen, h}
}

type pbkdf2KeyDeriver struct {
	iters  int
	keyLen int
	h      func() hash.Hash
}

func (d *pbkdf2KeyDeriver) DeriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key(
		[]byte(password),
		salt,
		d.iters,
		d.keyLen,
		d.h,
	)
}

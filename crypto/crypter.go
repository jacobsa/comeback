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

package blob

import (
	"fmt"
	"github.com/jacobsa/crypto/siv"
)

// A Crypter knows how to encrypt and decrypt arbitrary byte strings.
type Crypter interface {
	Encrypt(plaintext []byte) (ciphertext []byte, err error)
	Decrypt(ciphertext []byte) (plaintext []byte, err error)
}

// NotAuthenticError may be returned by Crypter.Decrypt if the input is
// otherwise well-formed but the ciphertext doesn't check out as authentic.
// This could be due to an incorrect key or corrupted ciphertext.
type NotAuthenticError string

func (e NotAuthenticError) Error() string {
	return string(e)
}

// Return a crypter configured to use AES-SIV deterministic decryption with
// authentication (see RFC 5297) with the supplied key. The key must be 32, 48,
// or 64 bytes long.
func NewCrypter(key []byte) (Crypter, error) {
	switch len(key) {
	case 32, 48, 64:
	default:
		return nil, fmt.Errorf("NewCrypter requires a 32-, 48-, or 64-byte key.")
	}

	return &sivCrypter{key}, nil
}

type sivCrypter struct {
	key []byte
}

func (c *sivCrypter) Encrypt(plaintext []byte) ([]byte, error) {
	return siv.Encrypt(c.key, plaintext, nil)
}

func (c *sivCrypter) Decrypt(ciphertext []byte) ([]byte, error) {
	return siv.Decrypt(c.key, ciphertext, nil)
}

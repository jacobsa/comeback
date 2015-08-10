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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/crypto"
)

// Return a blob store that wraps the supplied one, encrypting and decrypting
// data as it passes through. The supplied crypter should have deterministic
// output.
func NewEncryptingStore(crypter crypto.Crypter, wrapped Store) Store {
	return &encryptingStore{crypter, wrapped}
}

type encryptingStore struct {
	crypter crypto.Crypter
	wrapped Store
}

func (s *encryptingStore) Store(
	ctx context.Context,
	blob []byte) (Score, error) {
	// Encrypt the plaintext blob.
	ciphertext, err := s.crypter.Encrypt(blob)
	if err != nil {
		return Score{}, fmt.Errorf("Encrypt: %v", err)
	}

	// Pass on the encrypted blob.
	return s.wrapped.Store(ctx, ciphertext)
}

func (s *encryptingStore) Load(
	ctx context.Context,
	score Score) ([]byte, error) {
	// Load the encrypted blob.
	ciphertext, err := s.wrapped.Load(ctx, score)
	if err != nil {
		return nil, err
	}

	// Decrypt the ciphertext.
	plaintext, err := s.crypter.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("Decrypt: %v", err)
	}

	return plaintext, nil
}

func (s *encryptingStore) Flush(ctx context.Context) (err error) {
	err = s.wrapped.Flush(ctx)
	return
}

func (s *encryptingStore) Contains(ctx context.Context, score Score) (b bool) {
	b = s.wrapped.Contains(ctx, score)
	return
}

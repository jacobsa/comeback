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

package crypto_test

import (
	"github.com/jacobsa/comeback/crypto"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestCrypter(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type CrypterTest struct{}

func init() { RegisterTestSuite(&CrypterTest{}) }

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *CrypterTest) NilKey() {
	key := []byte(nil)

	_, err := crypto.NewCrypter(key)
	ExpectThat(err, Error(HasSubstr("-byte")))
}

func (t *CrypterTest) ShortKey() {
	key := make([]byte, 31)

	_, err := crypto.NewCrypter(key)
	ExpectThat(err, Error(HasSubstr("-byte")))
}

func (t *CrypterTest) LongKey() {
	key := make([]byte, 65)

	_, err := crypto.NewCrypter(key)
	ExpectThat(err, Error(HasSubstr("-byte")))
}

func (t *CrypterTest) Rfc5297TestCaseA1() {
	ExpectEq("TODO", "")
}

func (t *CrypterTest) CorruptedCiphertext() {
	ExpectEq("TODO", "")
}

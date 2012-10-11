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

package backup_test

import (
	"errors"
	"github.com/jacobsa/aws/sdb/mock"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestRegistry(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type registryTest struct {
	crypter mock_crypto.MockCrypter
	domain  mock_sdb.MockDomain
}

func (t *registryTest) SetUp(i *TestInfo) {
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.domain = mock_sdb.NewMockDomain(i.MockController, "domain")
}

////////////////////////////////////////////////////////////////////////
// NewRegistry
////////////////////////////////////////////////////////////////////////

type NewRegistryTest struct {
	registryTest

	r backup.Registry
	err error
}

func init() { RegisterTestSuite(&NewRegistryTest{}) }

func (t *NewRegistryTest) callConstructor() {
	t.r, t.err = backup.NewRegistry(t.crypter, t.domain)
}

func (t *NewRegistryTest) CallsGetAttributes() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(
		"comeback_marker",
		false,
		ElementsAre("encrypted_data")).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) GetAttributesReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("GetAttributes")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) CallsDecrypt() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) DecryptReturnsGenericError() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) DecryptReturnsNotAuthenticError() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) DecryptSucceeds() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsEncrypt() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) EncryptReturnsError() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsPutAttributes() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) PutAttributesReturnsError() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) PutAttributesSucceeds() {
	ExpectEq("TODO", "")
}

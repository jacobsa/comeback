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
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/aws/sdb/mock"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestRegistry(t *testing.T) { RunTests(t) }

const (
	domainName = "some_domain"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type registryTest struct {
	crypter mock_crypto.MockCrypter
	db mock_sdb.MockSimpleDB
	domain  mock_sdb.MockDomain
}

func (t *registryTest) SetUp(i *TestInfo) {
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.db = mock_sdb.NewMockSimpleDB(i.MockController, "domain")
	t.domain = mock_sdb.NewMockDomain(i.MockController, "domain")

	// By default, open the domain successfully.
	ExpectCall(t.db, "OpenDomain")(Any()).
		WillRepeatedly(oglemock.Return(t.domain, nil))
}

////////////////////////////////////////////////////////////////////////
// NewRegistry
////////////////////////////////////////////////////////////////////////

type NewRegistryTest struct {
	registryTest

	registry backup.Registry
	err error
}

func init() { RegisterTestSuite(&NewRegistryTest{}) }

func (t *NewRegistryTest) callConstructor() {
	t.registry, t.err = backup.NewRegistry(t.crypter, t.db, domainName)
}

func (t *NewRegistryTest) CallsOpenDomain() {
	// OpenDomain
	ExpectCall(t.db, "OpenDomain")(domainName).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) OpenDomainReturnsError() {
	// OpenDomain
	ExpectCall(t.db, "OpenDomain")(domainName).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("OpenDomain")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
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

func (t *NewRegistryTest) GetAttributesReturnsInvalidBase64Data() {
	// Domain
	attr := sdb.Attribute{Name: "encrypted_data", Value: "foo"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("base64")))
	ExpectThat(t.err, Error(HasSubstr("foo")))
}

func (t *NewRegistryTest) CallsDecrypt() {
	// Domain
	attr := sdb.Attribute{Name: "encrypted_data", Value: "dGFjbw=="}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(DeepEquals([]byte("taco"))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) DecryptReturnsGenericError() {
	// Domain
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("Decrypt")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) DecryptReturnsNotAuthenticError() {
	// Domain
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return(nil, &crypto.NotAuthenticError{}))

	// Call
	t.callConstructor()

	_, ok := t.err.(*backup.IncompatibleCrypterError)
	AssertTrue(ok, "Error: %v", t.err)

	ExpectThat(t.err, Error(HasSubstr("crypter")))
	ExpectThat(t.err, Error(HasSubstr("not compatible")))
}

func (t *NewRegistryTest) DecryptSucceeds() {
	// Domain
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Call
	t.callConstructor()

	AssertEq(nil, t.err)
	ExpectNe(nil, t.registry)
}

func (t *NewRegistryTest) CallsEncrypt() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) EncryptReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("Encrypt")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) CallsPutAttributes() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Crypter
	ciphertext := []byte("taco")
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(ciphertext, nil))

	// Domain
	expectedUpdate := sdb.PutUpdate{Name: "encrypted_data", Value: "dGFjbw=="}
	expectedPrecondition := sdb.Precondition{Name: "encrypted_data"}

	ExpectCall(t.domain, "PutAttributes")(
		"comeback_marker",
		ElementsAre(DeepEquals(expectedUpdate)),
		Pointee(DeepEquals(expectedPrecondition))).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) PutAttributesReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Domain
	ExpectCall(t.domain, "PutAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("PutAttributes")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) PutAttributesSucceeds() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Domain
	ExpectCall(t.domain, "PutAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	t.callConstructor()

	AssertEq(nil, t.err)
	ExpectNe(nil, t.registry)
}

////////////////////////////////////////////////////////////////////////
// RecordBackup
////////////////////////////////////////////////////////////////////////

type RecordBackupTest struct {
	registryTest
	registry backup.Registry

	job backup.CompletedJob
	err error
}

func init() { RegisterTestSuite(&RecordBackupTest{}) }

func (t *RecordBackupTest) SetUp(i *TestInfo) {
	var err error

	// Call common setup code.
	t.registryTest.SetUp(i)

	// Set up dependencies to pretend that the crypter is compatible.
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Create the registry.
	t.registry, err = backup.NewRegistry(t.crypter, t.db, domainName)
	AssertEq(nil, err)

	// Make the request legal by default.
	t.job.Name = "foo"
	t.job.Score = blob.ComputeScore([]byte{})
}

func (t *RecordBackupTest) callRegistry() {
	t.err = t.registry.RecordBackup(t.job)
}

func (t *RecordBackupTest) EmptyJobName() {
	t.job.Name = ""

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("job name")))
	ExpectThat(t.err, Error(HasSubstr("empty")))
}

func (t *RecordBackupTest) InvalidUtf8JobName() {
	ExpectEq("TODO", "")
}

func (t *RecordBackupTest) LongJobName() {
	ExpectEq("TODO", "")
}

func (t *RecordBackupTest) CallsPutAttributes() {
	ExpectEq("TODO", "")
}

func (t *RecordBackupTest) PutAttributesReturnsError() {
	ExpectEq("TODO", "")
}

func (t *RecordBackupTest) PutAttributesSucceeds() {
	ExpectEq("TODO", "")
}

////////////////////////////////////////////////////////////////////////
// ListRecentBackups
////////////////////////////////////////////////////////////////////////

type ListRecentBackupsTest struct {
	registryTest
	registry backup.Registry

	jobs []backup.CompletedJob
	err error
}

func init() { RegisterTestSuite(&ListRecentBackupsTest{}) }

func (t *ListRecentBackupsTest) SetUp(i *TestInfo) {
	var err error

	// Call common setup code.
	t.registryTest.SetUp(i)

	// Set up dependencies to pretend that the crypter is compatible.
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Create the registry.
	t.registry, err = backup.NewRegistry(t.crypter, t.db, domainName)
	AssertEq(nil, err)
}

func (t *ListRecentBackupsTest) callRegistry() {
	t.jobs, t.err = t.registry.ListRecentBackups()
}

func (t *ListRecentBackupsTest) CallsSelect() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) SelectReturnsError() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) NoResults() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) OneResultMissingName() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) OneResultMissingStartTime() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) OneResultMissingScore() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) OneResultHasInvalidScore() {
	ExpectEq("TODO", "")
}

func (t *ListRecentBackupsTest) ReturnsCompletedJobs() {
	ExpectEq("TODO", "")
}

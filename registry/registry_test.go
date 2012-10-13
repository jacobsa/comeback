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

package registry

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/aws/sdb/mock"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"strings"
	"testing"
	"time"
)

func TestRegistry(t *testing.T) { RunTests(t) }

const (
	domainName     = "some_domain"
	cryptoPassword = "some_password"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type registryTest struct {
	db      mock_sdb.MockSimpleDB
	domain  mock_sdb.MockDomain
	deriver mock_crypto.MockKeyDeriver
	crypter mock_crypto.MockCrypter
	randBytes *bytes.Buffer
}

func (t *registryTest) SetUp(i *TestInfo) {
	t.db = mock_sdb.NewMockSimpleDB(i.MockController, "domain")
	t.domain = mock_sdb.NewMockDomain(i.MockController, "domain")
	t.deriver = mock_crypto.NewMockKeyDeriver(i.MockController, "deriver")
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.randBytes = bytes.NewBuffer(make([]byte, 8))

	// Set up the domain's name and associated database.
	ExpectCall(t.domain, "Name")().
		WillRepeatedly(oglemock.Return(domainName))

	ExpectCall(t.domain, "Db")().
		WillRepeatedly(oglemock.Return(t.db))
}

type extentRegistryTest struct {
	registryTest
	registry Registry
}

func (t *extentRegistryTest) SetUp(i *TestInfo) {
	var err error

	// Call common setup code.
	t.registryTest.SetUp(i)

	// Set up a function that will return the mock crypter.
	createCrypter := func(key []byte) (crypto.Crypter, error) {
		return t.crypter, nil
	}

	// Set up dependencies to pretend that the crypter is compatible.
	attr := sdb.Attribute{Name: "encrypted_data"}
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{attr}, nil))

	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Create the registry.
	t.registry, err = newRegistry(
		t.domain,
		cryptoPassword,
		t.deriver,
		createCrypter,
		t.randBytes,
	)

	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// NewRegistry
////////////////////////////////////////////////////////////////////////

type NewRegistryTest struct {
	registryTest

	suppliedKey []byte
	createCrypterError error
	createCrypter func([]byte) (crypto.Crypter, error)

	registry Registry
	err      error
}

func init() { RegisterTestSuite(&NewRegistryTest{}) }

func (t *NewRegistryTest) SetUp(i *TestInfo) {
	// Call common setup code.
	t.registryTest.SetUp(i)

	// Set up the crypter factory function.
	t.createCrypter = func(key []byte) (crypto.Crypter, error) {
		t.suppliedKey = key
		return t.crypter, t.createCrypterError
	}
}

func (t *NewRegistryTest) callConstructor() {
	t.registry, t.err = newRegistry(
		t.domain,
		cryptoPassword,
		t.deriver,
		t.createCrypter,
		t.randBytes,
	)
}

func (t *NewRegistryTest) CallsGetAttributes() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(
		"comeback_marker",
		false,
		ElementsAre("encrypted_data", "password_salt")).
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

func (t *NewRegistryTest) MissingOnlyEncryptedDataAttribute() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) MissingOnlySaltAttribute() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) InvalidEncryptedDataAttribute() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "encrypted_data", Value: "foo"},
		sdb.Attribute{Name: "password_salt", Value: "YnVycml0bw=="},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("base64")))
	ExpectThat(t.err, Error(HasSubstr("foo")))
}

func (t *NewRegistryTest) InvalidSaltAttribute() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsDeriverAndCrypterFactoryForExistingMarkers() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsDecrypt() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "encrypted_data", Value: "dGFjbw=="},
		sdb.Attribute{Name: "password_salt", Value: "YnVycml0bw=="},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(DeepEquals([]byte("taco"))).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) DecryptReturnsError() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "encrypted_data", Value: ""},
		sdb.Attribute{Name: "password_salt", Value: ""},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("Decrypt")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) DecryptSucceeds() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "encrypted_data", Value: ""},
		sdb.Attribute{Name: "password_salt", Value: ""},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ExpectCall(t.crypter, "Decrypt")(Any()).
		WillOnce(oglemock.Return([]byte{}, nil))

	// Call
	t.callConstructor()

	AssertEq(nil, t.err)
	ExpectNe(nil, t.registry)
}

func (t *NewRegistryTest) ErrorGettingSaltBytes() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsDeriverAndCrypterFactoryForNewMarkers() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CrypterFactoryReturnsError() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) ErrorGettingDataBytes() {
	ExpectEq("TODO", "")
}

func (t *NewRegistryTest) CallsEncrypt() {
	someBytes := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}

	t.randBytes.Reset()
	t.randBytes.Write(someBytes)

	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(DeepEquals(someBytes[8:])).
		WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) EncryptReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callConstructor()

	ExpectThat(t.err, Error(HasSubstr("Encrypt")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *NewRegistryTest) CallsPutAttributes() {
	t.randBytes.Reset()
	t.randBytes.Write(make([]byte, 4))
	t.randBytes.Write([]byte("burrito!"))

	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

	// Crypter
	ciphertext := []byte("taco")
	ExpectCall(t.crypter, "Encrypt")(Any()).
		WillOnce(oglemock.Return(ciphertext, nil))

	// Domain
	expectedUpdates := []sdb.PutUpdate{
		sdb.PutUpdate{Name: "encrypted_data", Value: "dGFjbw=="},
		sdb.PutUpdate{Name: "password_salt", Value: "YnVycml0byE="},
	}

	expectedPrecondition := sdb.Precondition{Name: "encrypted_data"}

	ExpectCall(t.domain, "PutAttributes")(
		"comeback_marker",
		DeepEquals(expectedUpdates),
		Pointee(DeepEquals(expectedPrecondition))).
		WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callConstructor()
}

func (t *NewRegistryTest) PutAttributesReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return([]sdb.Attribute{}, nil))

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

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

	// Deriver
	ExpectCall(t.deriver, "DeriveKey")(Any(), Any()).
		WillOnce(oglemock.Return([]byte{}))

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
	extentRegistryTest

	job CompletedJob
	err error
}

func init() { RegisterTestSuite(&RecordBackupTest{}) }

func (t *RecordBackupTest) SetUp(i *TestInfo) {
	// Call common setup code.
	t.extentRegistryTest.SetUp(i)

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

	ExpectThat(t.err, Error(HasSubstr("Job name")))
	ExpectThat(t.err, Error(HasSubstr("empty")))
}

func (t *RecordBackupTest) InvalidUtf8JobName() {
	t.job.Name = "taco\x80\x81\x82"

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Job name")))
	ExpectThat(t.err, Error(HasSubstr("UTF-8")))
}

func (t *RecordBackupTest) LongJobName() {
	t.job.Name = strings.Repeat("x", 1025)

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Job name")))
	ExpectThat(t.err, Error(HasSubstr("1024")))
}

func (t *RecordBackupTest) CallsPutAttributes() {
	t.job.Id = 0xfeedface
	t.job.Name = "taco"
	t.job.StartTime = time.Date(1985, time.March, 18, 15, 33, 07, 0, time.UTC).Local()
	t.job.Score = blob.ComputeScore([]byte("burrito"))

	AssertThat(
		t.job.Score.Hex(),
		AllOf(
			MatchesRegexp("[a-f]"),
			MatchesRegexp("^[0-9a-f]{40}$"),
		),
	)

	// Domain
	ExpectCall(t.domain, "PutAttributes")(
		"backup_00000000feedface",
		ElementsAre(
			DeepEquals(
				sdb.PutUpdate{
					Name:  "job_name",
					Value: "taco",
				},
			),
			DeepEquals(
				sdb.PutUpdate{
					Name:  "start_time",
					Value: "1985-03-18T15:33:07Z",
				},
			),
			DeepEquals(
				sdb.PutUpdate{
					Name:  "score",
					Value: t.job.Score.Hex(),
				},
			),
		),
		Pointee(DeepEquals(sdb.Precondition{Name: "score", Value: nil})),
	).WillOnce(oglemock.Return(errors.New("")))

	// Call
	t.callRegistry()
}

func (t *RecordBackupTest) PutAttributesReturnsError() {
	// Domain
	ExpectCall(t.domain, "PutAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(errors.New("taco")))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("PutAttributes")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *RecordBackupTest) PutAttributesSucceeds() {
	// Domain
	ExpectCall(t.domain, "PutAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil))

	// Call
	t.callRegistry()

	ExpectEq(nil, t.err)
}

////////////////////////////////////////////////////////////////////////
// ListRecentBackups
////////////////////////////////////////////////////////////////////////

type ListRecentBackupsTest struct {
	extentRegistryTest

	jobs []CompletedJob
	err  error
}

func init() { RegisterTestSuite(&ListRecentBackupsTest{}) }

func (t *ListRecentBackupsTest) callRegistry() {
	t.jobs, t.err = t.registry.ListRecentBackups()
}

func (t *ListRecentBackupsTest) CallsSelect() {
	// Domain
	ExpectCall(t.db, "Select")(
		fmt.Sprintf(
			"select job_name, start_time, score "+
				"from `%s` where start_time is not null order by start_time desc",
			domainName),
		false,
		nil,
	).WillOnce(oglemock.Return(nil, nil, errors.New("")))

	// Call
	t.callRegistry()
}

func (t *ListRecentBackupsTest) SelectReturnsError() {
	// Domain
	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil, nil, errors.New("taco")))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Select")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *ListRecentBackupsTest) NoResults() {
	// Domain
	results := []sdb.SelectedItem{}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()
	AssertEq(nil, t.err)

	ExpectThat(t.jobs, ElementsAre())
}

func (t *ListRecentBackupsTest) OneResultHasJunkItemName() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "foobar",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("name")))
	ExpectThat(t.err, Error(HasSubstr("foobar")))
}

func (t *ListRecentBackupsTest) OneResultHasShortItemName() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedfa",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("name")))
	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedfa")))
}

func (t *ListRecentBackupsTest) OneResultHasLongItemName() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface0",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("name")))
	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface0")))
}

func (t *ListRecentBackupsTest) OneResultMissingJobName() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("name")))
}

func (t *ListRecentBackupsTest) OneResultMissingStartTime() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("start_time")))
}

func (t *ListRecentBackupsTest) OneResultHasInvalidStartTime() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "afsdf"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("invalid")))
	ExpectThat(t.err, Error(HasSubstr("start_time")))
	ExpectThat(t.err, Error(HasSubstr("afsdf")))
}

func (t *ListRecentBackupsTest) OneResultMissingScore() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("score")))
}

func (t *ListRecentBackupsTest) OneResultHasInvalidCharacterInScore() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 39) + "x"},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("invalid")))
	ExpectThat(t.err, Error(HasSubstr("score")))
	ExpectThat(t.err, Error(HasSubstr("fffx")))
}

func (t *ListRecentBackupsTest) OneResultHasShortScore() {
	validItem := sdb.SelectedItem{
		Name: "backup_00000000deadbeef",
		Attributes: []sdb.Attribute{
			sdb.Attribute{Name: "job_name", Value: "some_job"},
			sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
			sdb.Attribute{Name: "score", Value: strings.Repeat("f", 40)},
		},
	}

	// Domain
	results := []sdb.SelectedItem{
		validItem,
		sdb.SelectedItem{
			Name: "backup_00000000feedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "some_job"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: strings.Repeat("f", 39)},
			},
		},
		validItem,
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("backup_00000000feedface")))
	ExpectThat(t.err, Error(HasSubstr("invalid")))
	ExpectThat(t.err, Error(HasSubstr("score")))
	ExpectThat(t.err, Error(HasSubstr("fff")))
}

func (t *ListRecentBackupsTest) ReturnsCompletedJobs() {
	// Domain
	score0 := blob.ComputeScore([]byte("enchilada"))
	score1 := blob.ComputeScore([]byte("queso"))

	results := []sdb.SelectedItem{
		sdb.SelectedItem{
			Name: "backup_00000000deadbeef",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "job_name", Value: "taco"},
				sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
				sdb.Attribute{Name: "score", Value: score0.Hex()},
				sdb.Attribute{Name: "irrelevant", Value: "blah"},
			},
		},
		sdb.SelectedItem{
			Name: "backup_cafebabefeedface",
			Attributes: []sdb.Attribute{
				sdb.Attribute{Name: "irrelevant", Value: "blah"},
				sdb.Attribute{Name: "job_name", Value: "burrito"},
				sdb.Attribute{Name: "start_time", Value: "1985-03-18T15:33:07Z"},
				sdb.Attribute{Name: "score", Value: score1.Hex()},
			},
		},
	}

	ExpectCall(t.db, "Select")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(results, nil, nil))

	// Call
	t.callRegistry()
	AssertEq(nil, t.err)

	AssertThat(t.jobs, ElementsAre(Any(), Any()))

	ExpectEq(uint64(0xdeadbeef), t.jobs[0].Id)
	ExpectEq("taco", t.jobs[0].Name)
	ExpectThat(t.jobs[0].Score, DeepEquals(score0))
	ExpectTrue(
		time.Date(2012, time.August, 15, 12, 56, 00, 0, time.UTC).Local().Equal(
			t.jobs[0].StartTime),
		"Time: %v",
		t.jobs[0].StartTime,
	)

	ExpectEq(uint64(0xcafebabefeedface), t.jobs[1].Id)
	ExpectEq("burrito", t.jobs[1].Name)
	ExpectThat(t.jobs[1].Score, DeepEquals(score1))
	ExpectTrue(
		time.Date(1985, time.March, 18, 15, 33, 07, 0, time.UTC).Local().Equal(
			t.jobs[1].StartTime),
		"Time: %v",
		t.jobs[1].StartTime,
	)
}

////////////////////////////////////////////////////////////////////////
// FindBackup
////////////////////////////////////////////////////////////////////////

type FindBackupTest struct {
	extentRegistryTest

	jobId uint64
	job   CompletedJob
	err   error
}

func init() { RegisterTestSuite(&FindBackupTest{}) }

func (t *FindBackupTest) callRegistry() {
	t.job, t.err = t.registry.FindBackup(t.jobId)
}

func (t *FindBackupTest) CallsGetAttributes() {
	t.jobId = 0xdeadbeef

	// Domain
	ExpectCall(t.domain, "GetAttributes")(
		"backup_00000000deadbeef",
		false,
		ElementsAre("job_name", "start_time", "score"),
	).WillOnce(oglemock.Return(nil, errors.New("")))

	// Call
	t.callRegistry()
}

func (t *FindBackupTest) GetAttributesReturnsError() {
	// Domain
	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(nil, errors.New("taco")))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("GetAttributes")))
	ExpectThat(t.err, Error(HasSubstr("taco")))
}

func (t *FindBackupTest) JobNameMissing() {
	// Domain
	score := blob.ComputeScore([]byte(""))

	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
		sdb.Attribute{Name: "score", Value: score.Hex()},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("job_name")))
}

func (t *FindBackupTest) StartTimeMissing() {
	// Domain
	score := blob.ComputeScore([]byte(""))

	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "score", Value: score.Hex()},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("start_time")))
}

func (t *FindBackupTest) StartTimeInvalid() {
	// Domain
	score := blob.ComputeScore([]byte(""))

	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "asdsdfdg"},
		sdb.Attribute{Name: "score", Value: score.Hex()},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("invalid")))
	ExpectThat(t.err, Error(HasSubstr("start_time")))
	ExpectThat(t.err, Error(HasSubstr("asdsdfdg")))
}

func (t *FindBackupTest) ScoreMissing() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Missing")))
	ExpectThat(t.err, Error(HasSubstr("score")))
}

func (t *FindBackupTest) ScoreContainsIllegalCharacter() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
		sdb.Attribute{Name: "score", Value: strings.Repeat("f", 39) + "x"},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("score")))
	ExpectThat(t.err, Error(HasSubstr("fffx")))
}

func (t *FindBackupTest) ScoreTooLong() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
		sdb.Attribute{Name: "score", Value: strings.Repeat("f", 41)},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("score")))
	ExpectThat(t.err, Error(HasSubstr("fff")))
}

func (t *FindBackupTest) ScoreTooShort() {
	// Domain
	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
		sdb.Attribute{Name: "score", Value: strings.Repeat("f", 39)},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()

	ExpectThat(t.err, Error(HasSubstr("Invalid")))
	ExpectThat(t.err, Error(HasSubstr("score")))
	ExpectThat(t.err, Error(HasSubstr("fff")))
}

func (t *FindBackupTest) EverythingOkay() {
	t.jobId = 0xdeadbeef

	// Domain
	score := blob.ComputeScore([]byte("enchilada"))
	AssertThat(score.Hex(), MatchesRegexp("[a-f]"))
	AssertThat(score.Hex(), MatchesRegexp("[0-9a-f]{40}"))

	attrs := []sdb.Attribute{
		sdb.Attribute{Name: "irrelevant", Value: "foo"},
		sdb.Attribute{Name: "job_name", Value: "taco"},
		sdb.Attribute{Name: "start_time", Value: "2012-08-15T12:56:00Z"},
		sdb.Attribute{Name: "score", Value: score.Hex()},
		sdb.Attribute{Name: "irrelevant", Value: "bar"},
	}

	ExpectCall(t.domain, "GetAttributes")(Any(), Any(), Any()).
		WillOnce(oglemock.Return(attrs, nil))

	// Call
	t.callRegistry()
	AssertEq(nil, t.err)

	ExpectEq(uint64(0xdeadbeef), t.job.Id)
	ExpectEq("taco", t.job.Name)
	ExpectThat(t.job.Score, DeepEquals(score))
	ExpectTrue(
		time.Date(2012, time.August, 15, 12, 56, 00, 0, time.UTC).Local().Equal(
			t.job.StartTime),
		"Time: %v",
		t.job.StartTime,
	)
}

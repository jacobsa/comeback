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
	"fmt"
	"github.com/jacobsa/aws/sdb"
	"github.com/jacobsa/aws/sdb/mock"
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/crypto"
	"github.com/jacobsa/comeback/crypto/mock"
	. "github.com/jacobsa/oglematchers"
	"github.com/jacobsa/oglemock"
	. "github.com/jacobsa/ogletest"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestRegistry(t *testing.T) { RunTests(t) }

const (
	domainName = "some_domain"
)

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type registryTest struct {
	db      mock_sdb.MockSimpleDB
	domain  mock_sdb.MockDomain
	crypter mock_crypto.MockCrypter
	randSrc *rand.Rand
}

func (t *registryTest) SetUp(i *TestInfo) {
	t.db = mock_sdb.NewMockSimpleDB(i.MockController, "domain")
	t.domain = mock_sdb.NewMockDomain(i.MockController, "domain")
	t.crypter = mock_crypto.NewMockCrypter(i.MockController, "crypter")
	t.randSrc = rand.New(rand.NewSource(17))

	// Set up the domain's name and associated database.
	ExpectCall(t.domain, "Name")().
		WillRepeatedly(oglemock.Return(domainName))

	ExpectCall(t.domain, "Db")().
		WillRepeatedly(oglemock.Return(t.db))
}

type extentRegistryTest struct {
	registryTest
	registry backup.Registry
}

func (t *extentRegistryTest) SetUp(i *TestInfo) {
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
	t.registry, err = backup.NewRegistry(t.domain, t.crypter, t.randSrc)
	AssertEq(nil, err)
}

////////////////////////////////////////////////////////////////////////
// NewRegistry
////////////////////////////////////////////////////////////////////////

type NewRegistryTest struct {
	registryTest

	registry backup.Registry
	err      error
}

func init() { RegisterTestSuite(&NewRegistryTest{}) }

func (t *NewRegistryTest) callConstructor() {
	t.registry, t.err = backup.NewRegistry(t.domain, t.crypter, t.randSrc)
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
	extentRegistryTest

	job backup.CompletedJob
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

	jobs []backup.CompletedJob
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
	job backup.CompletedJob
	err  error
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
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) JobNameMissing() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) StartTimeMissing() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) StartTimeInvalid() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) ScoreMissing() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) ScoreContainsIllegalCharacter() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) ScoreTooLong() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) ScoreTooShort() {
	ExpectEq("TODO", "")
}

func (t *FindBackupTest) EverythingOkay() {
	ExpectEq("TODO", "")
}

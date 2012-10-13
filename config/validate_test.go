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

package config_test

import (
	"github.com/jacobsa/aws"
	"github.com/jacobsa/comeback/config"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
	"testing"
)

func TestValidate(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ValidateTest struct {
	cfg *config.Config
}

func init() { RegisterTestSuite(&ValidateTest{}) }

func (t *ValidateTest) SetUp(i *TestInfo) {
	// Make the config valid by default.
	t.cfg = &config.Config{
		Jobs:      make(map[string]*config.Job),
		S3Bucket:  "foo",
		S3Region:  "foo",
		SdbDomain: "foo",
		SdbRegion: "foo",
		AccessKey: aws.AccessKey{
			Id:     "foo",
			Secret: "foo",
		},
	}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ValidateTest) JobNameNotValidUtf8() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "/a"}
	t.cfg.Jobs["foo\x80\x81\x82bar"] = &config.Job{BasePath: "/b"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("name")))
	ExpectThat(err, Error(HasSubstr("UTF-8")))
}

func (t *ValidateTest) EmptyBasePath() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: ""}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("path")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ValidateTest) BasePathNotAbsolute() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "a"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("path")))
	ExpectThat(err, Error(HasSubstr("absolute")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ValidateTest) BasePathNotValidUtf8() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "/a/\x80\x81\x82"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("path")))
	ExpectThat(err, Error(HasSubstr("UTF-8")))
	ExpectThat(err, Error(HasSubstr("taco")))
}

func (t *ValidateTest) MissingS3Bucket() {
	t.cfg.S3Bucket = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("S3")))
	ExpectThat(err, Error(HasSubstr("bucket")))
}

func (t *ValidateTest) MissingS3Region() {
	t.cfg.S3Region = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("S3")))
	ExpectThat(err, Error(HasSubstr("region")))
}

func (t *ValidateTest) MissingSdbDomain() {
	t.cfg.SdbDomain = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("SimpleDB")))
	ExpectThat(err, Error(HasSubstr("domain")))
}

func (t *ValidateTest) MissingSdbRegion() {
	t.cfg.SdbRegion = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("SimpleDB")))
	ExpectThat(err, Error(HasSubstr("region")))
}

func (t *ValidateTest) MissingAccessKeyId() {
	t.cfg.AccessKey.Id = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("AWS")))
	ExpectThat(err, Error(HasSubstr("key ID")))
}

func (t *ValidateTest) MissingAccessKeySecret() {
	t.cfg.AccessKey.Secret = ""

	err := config.Validate(t.cfg)

	ExpectThat(err, Error(HasSubstr("AWS")))
	ExpectThat(err, Error(HasSubstr("key secret")))
}

func (t *ValidateTest) EverythingValid() {
	err := config.Validate(t.cfg)
	ExpectEq(nil, err)
}

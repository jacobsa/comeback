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
	t.cfg = &config.Config{Jobs: make(map[string]*config.Job)}
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ValidateTest) JobNameNotValidUtf8() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "/a"}
	t.cfg.Jobs["foo\x80\x81\x82bar"] = &config.Job{BasePath: "/b"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, HasSubstr("name"))
	ExpectThat(err, HasSubstr("UTF-8"))
}

func (t *ValidateTest) EmptyBasePath() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: ""}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, HasSubstr("base path"))
	ExpectThat(err, HasSubstr("taco"))
}

func (t *ValidateTest) BasePathNotAbsolute() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "a"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, HasSubstr("path"))
	ExpectThat(err, HasSubstr("absolute"))
	ExpectThat(err, HasSubstr("taco"))
}

func (t *ValidateTest) BasePathNotValidUtf8() {
	t.cfg.Jobs["taco"] = &config.Job{BasePath: "/a/\x80\x81\x82"}
	t.cfg.Jobs["burrito"] = &config.Job{BasePath: "/c"}

	err := config.Validate(t.cfg)

	ExpectThat(err, HasSubstr("path"))
	ExpectThat(err, HasSubstr("UTF-8"))
	ExpectThat(err, HasSubstr("taco"))
}

func (t *ValidateTest) EverythingValid() {
	ExpectEq("TODO", "")
}

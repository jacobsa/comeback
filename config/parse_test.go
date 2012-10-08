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

func TestParse(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

type ParseTest struct {
	data string
	cfg *config.Config
	err error
}

func init() { RegisterTestSuite(&ParseTest{}) }

func (t *ParseTest) parse() {
	t.cfg, t.err = config.Parse([]byte(t.data))
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ParseTest) TotalJunk() {
	t.data = "sdhjklfghdskjghdjkfgj"
	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) NonObject() {
	t.data = `[17, 19]`
	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) MissingTrailingBrace() {
	t.data = `
	{
		"jobs": []
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) BasePathIsNumber() {
	t.data = `
	{
		"jobs": [
			{
				"base_path": 17
			}
		]
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) BasePathIsNull() {
	t.data = `
	{
		"jobs": [
			{
				"base_path": null
			}
		]
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) BasePathIsObject() {
	t.data = `
	{
		"jobs": [
			{
				"base_path": {}
			}
		]
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) OneExcludeDoesntCompile() {
	t.data = `
	{
		"jobs": [
			{
				"base_path": "/foo",
				"excludes": ["a"],
			},
			{
				"base_path": "/bar",
				"excludes": ["b", "(c"]
			},
			{
				"base_path": "/foo",
				"excludes": ["d"]
			}
		]
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) EmptyConfig() {
	t.data = `{}`
	t.parse()

	AssertEq(nil, t.err)
	ExpectNe(nil, t.cfg.Jobs)
	ExpectEq(0, len(t.cfg.Jobs))
}

func (t *ParseTest) MissingExcludesArray() {
	ExpectEq("TODO", "")
}

func (t *ParseTest) DuplicateJobName() {
	ExpectEq("TODO", "")
}

func (t *ParseTest) StructurallyValid() {
	ExpectEq("TODO", "")
}

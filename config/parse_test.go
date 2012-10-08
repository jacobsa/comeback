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
	cfg  *config.Config
	err  error
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

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("invalid")))
}

func (t *ParseTest) Null() {
	t.data = `null`
	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("null")))
}

func (t *ParseTest) Array() {
	t.data = `[17, 19]`
	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("array")))
}

func (t *ParseTest) MissingTrailingBrace() {
	t.data = `
	{
		"jobs": {}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("unexpected end")))
}

func (t *ParseTest) BasePathIsNumber() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": 17
			}
		}
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("number")))
}

func (t *ParseTest) BasePathIsNull() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": null
			}
		}
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("null")))
}

func (t *ParseTest) BasePathIsObject() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": {}
			}
		}
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("JSON")))
	ExpectThat(t.err, Error(HasSubstr("object")))
}

func (t *ParseTest) OneExcludeDoesntCompile() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": "/foo",
				"excludes": ["a"]
			},
			"burrito": {
				"base_path": "/bar",
				"excludes": ["b", "(c"]
			},
			"enchilada": {
				"base_path": "/foo",
				"excludes": ["d"]
			}
		}
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
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": "/foo"
			}
		}
	}
	`

	t.parse()

	AssertEq(nil, t.err)
	AssertEq(1, len(t.cfg.Jobs))

	AssertNe(nil, t.cfg.Jobs["taco"])
	ExpectThat(t.cfg.Jobs["taco"].Excludes, ElementsAre())
}

func (t *ParseTest) DuplicateJobName() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": "/foo"
			},
			"burrito": {
				"base_path": "/bar"
			},
			"taco": {
				"base_path": "/enchilada"
			}
		}
	}
	`

	t.parse()

	ExpectThat(t.err, Error(HasSubstr("TODO")))
}

func (t *ParseTest) MultipleValidJobs() {
	t.data = `
	{
		"jobs": {
			"taco": {
				"base_path": "/foo"
				"excludes": ["a.b"],
			},
			"burrito": {
				"base_path": "/bar",
				"excludes": ["c", "d"]
			}
		}
	}
	`

	t.parse()

	AssertEq(nil, t.err)
	AssertEq(2, len(t.cfg.Jobs))

	AssertNe(nil, t.cfg.Jobs["taco"])
	ExpectEq("/foo", t.cfg.Jobs["taco"].BasePath)
	AssertThat(t.cfg.Jobs["taco"].Excludes, ElementsAre(Any()))
	ExpectEq("a.b", t.cfg.Jobs["taco"].Excludes[0])

	AssertNe(nil, t.cfg.Jobs["burrito"])
	ExpectEq("/bar", t.cfg.Jobs["burrito"].BasePath)
	AssertThat(t.cfg.Jobs["burrito"].Excludes, ElementsAre(Any(), Any()))
	ExpectEq("c", t.cfg.Jobs["burrito"].Excludes[0])
	ExpectEq("d", t.cfg.Jobs["burrito"].Excludes[1])
}

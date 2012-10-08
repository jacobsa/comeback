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

package config

import (
	"fmt"
	"unicode/utf8"
)

func validateJob(j *Job) error {
	// Base paths must be non-empty valid UTF-8.
	if j.BasePath == "" || !utf8.Valid([]byte(j.BasePath)) {
		return fmt.Errorf("Base paths must be non-empty valid UTF-8.")
	}

	return nil
}

// Return an error if the supplied config data is invalid in some way.
func Validate(c *Config) error {
	// Check each job.
	for name, job := range c.Jobs {
		// Names must be valid UTF-8.
		if !utf8.Valid([]byte(name)) {
			return fmt.Errorf("Job names must be valid UTF-8.")
		}

		// Check the job itself.
		if err := validateJob(job); err != nil {
			return fmt.Errorf("Job %s: %v", name, err)
		}
	}

	return nil
}

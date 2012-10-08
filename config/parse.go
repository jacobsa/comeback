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
	"encoding/json"
	"fmt"
	"regexp"
)

type jsonJob struct {
	BasePath string   `json:"base_path"`
	Excludes []string `json:"excludes"`
}

type jsonConfig struct {
	Jobs map[string]*jsonJob `json:"jobs"`
}

// Parse the supplied JSON configuration data.
func Parse(data []byte) (*Config, error) {
	// Parse the JSON into our private representation.
	var jCfg jsonConfig
	if err := json.Unmarshal(data, &jCfg); err != nil {
		return nil, fmt.Errorf("Decoding JSON: %v", err)
	}

	// Convert to our public representation.
	cfg := &Config{Jobs: make(map[string]*Job)}
	for name, jJob := range jCfg.Jobs {
		// Create a public job and populate it.
		job := new(Job)
		cfg.Jobs[name] = job

		job.BasePath = jJob.BasePath
		for _, reStr := range jJob.Excludes {
			re, err := regexp.Compile(reStr)
			if err != nil {
				return nil, err
			}

			job.Excludes = append(job.Excludes, re)
		}
	}

	return cfg, nil
}
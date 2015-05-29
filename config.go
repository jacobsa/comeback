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

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path"
	"sync"

	"github.com/jacobsa/comeback/config"
)

var g_configOnce sync.Once
var g_config *config.Config

func getConfigFilePath() (p string, err error) {
	// Find the user.
	u, err := user.Current()
	if err != nil {
		err = fmt.Errorf("user.Current: %v", err)
		return
	}

	// Use the file within her homedir.
	p = path.Join(u.HomeDir, ".comeback.json")

	return
}

func initConfig() {
	var err error

	// Find the file's path.
	filename, err := getConfigFilePath()
	if err != nil {
		log.Fatalf("getConfigFilePath: %v", err)
		return
	}

	// Read the file.
	configData, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln("Error reading config file:", err)
	}

	// Parse the config file.
	g_config, err = config.Parse(configData)
	if err != nil {
		log.Fatalln("Parsing config file:", err)
	}

	// Validate.
	if err := config.Validate(g_config); err != nil {
		log.Fatalln("Invalid config:", err)
	}
}

func getConfig() *config.Config {
	g_configOnce.Do(initConfig)
	return g_config
}

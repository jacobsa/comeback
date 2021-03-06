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
	"log"
	"os"
	"sync"

	"github.com/jacobsa/util/password"
)

var gPassword string
var gPasswordOnce sync.Once

// If set, the user will not be prompted.
const passwordEnvVar = "COMEBACK_PASSWORD"

func initPassword() {
	// Is the environment variable set?
	var ok bool
	if gPassword, ok = os.LookupEnv(passwordEnvVar); ok {
		return
	}

	// Prompt the user.
	gPassword = password.ReadPassword("Enter crypto password: ")
	if len(gPassword) == 0 {
		log.Fatalln("You must enter a password.")
	}
}

func getPassword() string {
	gPasswordOnce.Do(initPassword)
	return gPassword
}

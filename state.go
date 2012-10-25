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
	"github.com/jacobsa/comeback/state"
	"log"
	"os"
	"sync"
)

var g_stateOnce sync.Once
var g_state state.State

func initState() {
	cfg := getConfig()
	var err error

	// Open the specified file.
	f, err := os.Open(cfg.StateFile)
	if err != nil {
		log.Fatalln("Opening state file:", err)
	}

	defer f.Close()

	// Load the state struct.
	g_state, err = state.LoadState(f)
	if err != nil {
		log.Fatalln("LoadState:", err)
	}

	// Throw out the existing scores set if it's out of date.
	reg := getRegistry()
	currentVersion, err := reg.GetCurrentScoreSetVersion()
	if err != nil {
		log.Fatalln("GetCurrentScoreSetVersion:", err)
	}

	if g_state.ExistingScoresVersion != currentVersion {
		g_state.ExistingScores = nil
		g_state.ExistingScoresVersion = currentVersion
	}
}

func getState() *state.State {
	g_stateOnce.Do(initState)
	return &g_state
}

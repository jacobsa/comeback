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
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/jacobsa/comeback/state"
)

var g_stateOnce sync.Once
var g_state state.State

var g_saveStateMutex sync.Mutex

func initState() {
	cfg := getConfig()
	var err error

	// Open the specified file.
	f, err := os.Open(cfg.StateFile)

	// Special case: if the error is that the file doesn't exist, ignore it.
	if err != nil && os.IsNotExist(err) {
		err = nil
		log.Println("No state file found. Using fresh state.")
	} else {
		// Handle other Open errors.
		if err != nil {
			log.Fatalln("Opening state file:", err)
		}

		defer f.Close()

		// Load the state struct.
		log.Println("Loading from state file.")

		g_state, err = state.LoadState(f)
		if err != nil {
			log.Fatalln("LoadState:", err)
		}
	}

	// Throw out the existing score cache if requested.
	if *g_discardScoreCache {
		g_state.ScoresForFiles = state.NewScoreMap()
	}

	// Make sure there are no nil interface values.
	if g_state.ScoresForFiles == nil {
		g_state.ScoresForFiles = state.NewScoreMap()
	}
}

func getState() *state.State {
	g_stateOnce.Do(initState)
	return &g_state
}

func saveState() {
	// Make sure only one run can happen at a time.
	g_saveStateMutex.Lock()
	defer g_saveStateMutex.Unlock()

	var err error

	cfg := getConfig()
	stateStruct := getState()

	// Write out the state to a temporary file.
	f, err := ioutil.TempFile("", "comeback_state")
	if err != nil {
		log.Fatalln("Creating temporary state file:", err)
	}

	tempFilePath := f.Name()

	if err = state.SaveState(f, *stateStruct); err != nil {
		log.Fatalln("SaveState:", err)
	}

	if err = f.Close(); err != nil {
		log.Fatalln("Closing temporary state file:", err)
	}

	// Rename the file into the new location.
	if err = os.Rename(tempFilePath, cfg.StateFile); err != nil {
		log.Fatalln("Renaming temporary state file:", err)
	}
}

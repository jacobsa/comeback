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
	"os"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/state"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/comeback/internal/wiring"
	"github.com/jacobsa/gcloud/gcs"
)

var g_stateOnce sync.Once
var g_state state.State

var g_saveStateMutex sync.Mutex

func buildExistingScores(
	ctx context.Context,
	bucket gcs.Bucket) (existingScores util.StringSet, err error) {
	// List into a slice.
	slice, err := listAllScores(
		ctx,
		bucket,
		wiring.BlobObjectNamePrefix)

	if err != nil {
		err = fmt.Errorf("listAllScores: %v", err)
		return
	}

	// Build a set.
	existingScores = util.NewStringSet()
	for _, score := range slice {
		existingScores.Add(score.Hex())
	}

	return
}

func makeState(ctx context.Context) (s state.State, err error) {
	cfg := getConfig()
	bucket := getBucket(ctx)

	// Open the specified file.
	f, err := os.Open(cfg.StateFile)
	switch {
	// Special case: if the error is that the file doesn't exist, ignore it.
	case os.IsNotExist(err):
		err = nil
		log.Println("No state file found. Using fresh state.")

	case err != nil:
		return
	}

	// If we opened a file above, load from it.
	if f != nil {
		defer f.Close()
		s, err = state.LoadState(f)
		if err != nil {
			err = fmt.Errorf("LoadState: %v", err)
			return
		}
	}

	// Throw out the existing score cache if requested.
	if *g_discardScoreCache {
		s.ScoresForFiles = state.NewScoreMap()
	}

	// Make sure there are no nil interface values.
	if s.ScoresForFiles == nil {
		s.ScoresForFiles = state.NewScoreMap()
	}

	// If we don't know the set of hex scores in the store, or the set of scores
	// is stale, re-list.
	age := time.Now().Sub(s.RelistTime)
	const maxAge = 30 * 24 * time.Hour

	if s.ExistingScores == nil || age > maxAge {
		log.Println("Listing existing scores...")

		s.RelistTime = time.Now()
		s.ExistingScores, err = buildExistingScores(ctx, bucket)
		if err != nil {
			err = fmt.Errorf("buildExistingScores: %v", err)
			return
		}
	}

	return
}

func initState(ctx context.Context) {
	var err error

	// Load the state struct.
	log.Println("Loading from state file...")

	g_state, err = makeState(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	// Save it back to the file, in case makeState changed it (e.g. by listing
	// existing scores).
	log.Println("Saving to state file...")

	err = saveStateStruct(getConfig().StateFile, &g_state)
	if err != nil {
		log.Fatalf("saveStateStruct: %v", err)
	}

	log.Println("Finished saving to state file.")
}

func getState(ctx context.Context) *state.State {
	g_stateOnce.Do(func() { initState(ctx) })
	return &g_state
}

func saveStateStruct(dst string, s *state.State) (err error) {
	// Create a temporary file.
	f, err := ioutil.TempFile("", "comeback_state")
	if err != nil {
		err = fmt.Errorf("TempFile: %v", err)
		return
	}

	defer f.Close()
	tempFilePath := f.Name()

	// Write to the file.
	err = state.SaveState(f, *s)
	if err != nil {
		err = fmt.Errorf("SaveState: %v", err)
		return
	}

	// Close the file.
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("Close: %v", err)
		return
	}

	// Rename the file into the new location.
	err = os.Rename(tempFilePath, dst)
	if err != nil {
		err = fmt.Errorf("Rename: %v", err)
		return
	}

	return
}

func saveState(ctx context.Context) {
	// Make sure only one run can happen at a time.
	g_saveStateMutex.Lock()
	defer g_saveStateMutex.Unlock()

	cfg := getConfig()
	stateStruct := getState(ctx)

	err := saveStateStruct(cfg.StateFile, stateStruct)
	if err != nil {
		log.Fatalf("saveStateStruct: %v", err)
	}
}

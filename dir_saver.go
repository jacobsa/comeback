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

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/backup"
	"github.com/jacobsa/comeback/internal/wiring"
)

var gDirSaverOnce sync.Once
var gDirSaver backup.DirectorySaver

func initDirSaver(ctx context.Context) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	bucket := getBucket(ctx)
	password := getPassword()
	state := getState(ctx)

	const chunkSize = 1 << 24 // 16 MiB
	gDirSaver, err = wiring.MakeDirSaver(
		ctx,
		password,
		bucket,
		chunkSize,
		state.ExistingScores,
		state.ScoresForFiles,
		log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lmicroseconds))
}

func getDirSaver(ctx context.Context) backup.DirectorySaver {
	gDirSaverOnce.Do(func() { initDirSaver(ctx) })
	return gDirSaver
}

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
	"sync"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/backup"
	"github.com/jacobsa/comeback/internal/wiring"
)

var gDirRestorer backup.DirectoryRestorer
var gDirRestorerOnce sync.Once

func initDirRestorer(ctx context.Context) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	bucket := getBucket(ctx)
	password := getPassword()

	gDirRestorer, err = wiring.MakeDirRestorer(
		password,
		bucket)
}

func getDirRestorer(ctx context.Context) backup.DirectoryRestorer {
	gDirRestorerOnce.Do(func() { initDirRestorer(ctx) })
	return gDirRestorer
}

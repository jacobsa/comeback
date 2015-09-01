// Copyright 2015 Aaron Jacobs. All Rights Reserved.
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

package wiring

import (
	"fmt"

	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/sys"
)

// Create a FileSystem that writes to the real file system.
func makeFileSystem() (fileSystem fs.FileSystem, err error) {
	userRegistry := sys.NewUserRegistry()
	groupRegistry := sys.NewGroupRegistry()

	// Create the file system.
	fileSystem, err = fs.NewFileSystem(userRegistry, groupRegistry)
	if err != nil {
		err = fmt.Errorf("NewFileSystem: %v", err)
		return
	}

	return
}

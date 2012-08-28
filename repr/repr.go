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

// Package repr contains functions useful for serializing and deserializing
// directory listing structs.
package repr

import "fmt"
import "github.com/jacobsa/comeback/fs"

// Marshal turns a list of directory entries into a string that can later be
// used with Unmarshal.
func Marshal(entries []fs.DirectoryEntry) (s string, err error) {
	return "TODO", nil
}

// Unmarshal recovers a list of directory entries from a string previously
// created by Marshal.
func Unmarshal(s string) (entries []fs.DirectoryEntry, err error) {
	return nil, fmt.Errorf("TODO")
}

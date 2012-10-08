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

// Package disk implements a key/value store on the local disk.
package disk

import (
	"fmt"
	"github.com/jacobsa/comeback/kv"
	"github.com/jacobsa/comeback/fs"
	"io/ioutil"
	"path"
)

// Return a key/value store that stores its values in the directory with the
// supplied path. Only keys that represent valid file names are supported.
func NewDiskKvStore(path string, fileSystem fs.FileSystem) (kv.Store, error) {
	return &kvStore{basePath: path, fileSystem: fileSystem}, nil
}

type kvStore struct {
	basePath   string
	fileSystem fs.FileSystem
}

func (s *kvStore) Set(key []byte, val []byte) error {
	filePath := path.Join(s.basePath, string(key))

	if err := s.fileSystem.WriteFile(filePath, val, 0600); err != nil {
		return fmt.Errorf("WriteFile: %v", err)
	}

	return nil
}

func (s *kvStore) Get(key []byte) ([]byte, error) {
	filePath := path.Join(s.basePath, string(key))
	file, err := s.fileSystem.OpenForReading(filePath)
	if err != nil {
		return nil, fmt.Errorf("OpenForReading: %v", err)
	}

	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll: %v", err)
	}

	return data, nil
}

func (s *kvStore) Contains(key []byte) (res bool, err error) {
	// Don't bother trying. This is legal according to the interface.
	return false, nil
}

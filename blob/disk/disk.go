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

// Package disk implements a blob store on the local disk.
package disk

import (
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"io/ioutil"
	"path"
)

// Return a blob store that stores its blobs in the directory with the supplied
// path.
func NewDiskBlobStore(path string, fileSystem fs.FileSystem) (blob.Store, error) {
	return &blobStore{basePath: path, fileSystem: fileSystem}, nil
}

type blobStore struct {
	basePath   string
	fileSystem fs.FileSystem
}

func (s *blobStore) Store(b []byte) (blob.Score, error) {
	score := blob.ComputeScore(b)
	filePath := path.Join(s.basePath, score.Hex())

	if err := s.fileSystem.WriteFile(filePath, b, 0600); err != nil {
		return nil, fmt.Errorf("WriteFile: %v", err)
	}

	return score, nil
}

func (s *blobStore) Load(score blob.Score) ([]byte, error) {
	filePath := path.Join(s.basePath, score.Hex())
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

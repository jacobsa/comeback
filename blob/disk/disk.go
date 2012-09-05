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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"io/ioutil"
	"path"
)

type blobStore struct {
	basePath   string
	fileSystem fs.FileSystem
}

func (s *blobStore) Store(b []byte) (blob.Score, error) {
	score := blob.ComputeScore(b)
	filePath := path.Join(s.basePath, score.Hex())

	if err := ioutil.WriteFile(filePath, b, 0600); err != nil {
		return nil, err
	}

	return score, nil
}

func (s *blobStore) Load(score blob.Score) ([]byte, error) {
	filePath := path.Join(s.basePath, score.Hex())

	return ioutil.ReadFile(filePath)
}

// Return a blob store that stores its blobs in the directory with the supplied
// path.
func NewDiskBlobStore(path string, fileSystem fs.FileSystem) (blob.Store, error) {
	return &blobStore{basePath: path, fileSystem: fileSystem}, nil
}
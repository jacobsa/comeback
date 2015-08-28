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

package restore

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
)

// Create a dag.Visitor for *node.
//
// For each node n, the visitor does the following:
//
//  *  Ensure that the directory path.Dir(n.RelPath) exists.
//  *  <Perform type-specific action.>
//  *  Set the appropriate permissions and times for n.RelPath.
//
// Restoring ownership is not supported.
//
// The type-specific actions are as follows:
//
//  *  Files: create the file with the contents described by n.Info.Scores.
//  *  Directories: ensure that the directory n.RelPath exists.
//  *  Symlinks: create a symlink pointing at n.Info.Target.
//
func newVisitor(
	basePath string,
	blobStore blob.Store,
	logger *log.Logger) (v dag.Visitor) {
	v = &visitor{
		basePath:  basePath,
		blobStore: blobStore,
		logger:    logger,
	}

	return
}

type visitor struct {
	basePath  string
	blobStore blob.Store
	logger    *log.Logger
}

func (v *visitor) Visit(ctx context.Context, untyped dag.Node) (err error) {
	// Ensure the input is of the correct type.
	n, ok := untyped.(*node)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", untyped)
		return
	}

	absPath := path.Join(v.basePath, n.RelPath)

	// Make sure the leading directories exist so that we can write into them.
	err = os.MkdirAll(path.Dir(absPath), 0700)
	if err != nil {
		err = fmt.Errorf("MkdirAll: %v", err)
		return
	}

	// Perform type-specific logic.
	switch n.Info.Type {
	case fs.TypeSymlink:
		err = v.handleSymlink(n)
		if err != nil {
			err = fmt.Errorf("handleSymlink: %v", err)
			return
		}

	default:
		err = fmt.Errorf("Unhandled type %d for node: %q", n.Info.Type, n.RelPath)
		return
	}

	err = errors.New("TODO")
	return
}

func (v *visitor) handleSymlink(n *node) (err error) {
	err = errors.New("TODO")
	return
}

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
	"fmt"
	"log"
	"path"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr"
)

// Create a dag.DependencyResolver for *node.
//
// For directories, dependencies are resolved by loading a listing from
// n.Info.Scores[0], which must exist and be the only score. No other nodes
// have dependencies.
//
// Child nodes returned are filled into node.Children fields.
func newDependencyResolver(
	blobStore blob.Store,
	logger *log.Logger) (dr dag.DependencyResolver) {
	dr = &dependencyResolver{
		blobStore: blobStore,
		logger:    logger,
	}

	return
}

type dependencyResolver struct {
	blobStore blob.Store
	logger    *log.Logger
}

func (dr *dependencyResolver) FindDependencies(
	ctx context.Context,
	untyped dag.Node) (deps []dag.Node, err error) {
	// Ensure the input is of the correct type.
	n, ok := untyped.(*node)
	if !ok {
		err = fmt.Errorf("Node has unexpected type: %T", untyped)
		return
	}

	// Non-directories have no dependencies.
	if n.Info.Type != fs.TypeDirectory {
		return
	}

	// We expect exactly one score for the listing.
	if len(n.Info.Scores) != 1 {
		err = fmt.Errorf(
			"Unexpected score count for %q: %d",
			n.RelPath,
			len(n.Info.Scores))

		return
	}

	score := n.Info.Scores[0]

	// Load the listing blob.
	b, err := dr.blobStore.Load(ctx, score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", score.Hex(), err)
		return
	}

	// Parse the listing.
	listing, err := repr.UnmarshalDir(b)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir(%s): %v", score.Hex(), err)
		return
	}

	// Fill in the node's children and dependencies.
	for _, entry := range listing {
		child := &node{
			RelPath: path.Join(n.RelPath, entry.Name),
			Info:    *entry,
		}

		n.Children = append(n.Children, child)
		deps = append(deps, child)
	}

	return
}

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
	"log"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
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
	err = errors.New("TODO")
	return
}

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

package save

import (
	"fmt"
	"os"
	"path"

	"golang.org/x/net/context"
)

// Fill in fsNode.Info fields for all incoming nodes. Nodes are interpreted as
// relative to the supplied base path.
func statNodes(
	ctx context.Context,
	basePath string,
	nodesIn <-chan *fsNode,
	nodesOut chan<- *fsNode) (err error) {
	for n := range nodesIn {
		// Stat.
		n.Info, err = os.Stat(path.Join(basePath, n.RelPath))
		if err != nil {
			err = fmt.Errorf("Stat: %v", err)
			return
		}

		// Write to output.
		select {
		case nodesOut <- n:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

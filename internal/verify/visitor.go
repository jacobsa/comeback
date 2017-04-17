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

package verify

import (
	"context"
	"fmt"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/timeutil"
)

// Create a visitor that confirms file chunks can be loaded from the supplied
// blob store, writing out appropriate records to certify this. Nothing is done
// for directory nodes (which are handled by the dependency resolver).
//
// It is expected that the blob store's Load method does score verification for
// us.
func newVisitor(
	records chan<- Record,
	bs blob.Store,
	clock timeutil.Clock) (v dag.Visitor) {
	v = &visitor{
		records:   records,
		blobStore: bs,
		clock:     clock,
	}

	return
}

type visitor struct {
	records   chan<- Record
	blobStore blob.Store
	clock     timeutil.Clock
}

func (v *visitor) Visit(ctx context.Context, untyped dag.Node) (err error) {
	// Make sure the node is of the appropriate type.
	n, ok := untyped.(Node)
	if !ok {
		err = fmt.Errorf("Unexpected node type: %T", untyped)
		return
	}

	// There is nothing to do for directories.
	if n.Dir {
		return
	}

	// Make sure we can load the blob contents. We rely on the blob store to
	// verify the content against the score.
	_, err = v.blobStore.Load(ctx, n.Score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", n.Score.Hex(), err)
		return
	}

	// Certify that we verified the file chunk.
	r := Record{
		Time: v.clock.Now(),
		Node: n,
	}

	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case v.records <- r:
	}

	return
}

// A visitor that does nothing for each node.
type doNothingVisitor struct {
}

func (v *doNothingVisitor) Visit(context.Context, dag.Node) error {
	return nil
}

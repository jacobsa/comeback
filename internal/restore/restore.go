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
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/crypto"
	"github.com/jacobsa/comeback/internal/dag"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/sys"
	"github.com/jacobsa/comeback/internal/util"
	"github.com/jacobsa/gcloud/gcs"
)

// Restore the backup rooted at the supplied score into the given directory,
// which must already exist.
//
// The supplied bucket is assumed to contain objects with the given name
// prefix.
func Restore(
	ctx context.Context,
	dir string,
	score blob.Score,
	bucket gcs.Bucket,
	objectNamePrefix string,
	crypter crypto.Crypter,
	logger *log.Logger) (err error) {
	// Hopefully enough parallelism to keep our CPUs saturated (for decryption,
	// SHA-1 computation, etc.) or our NIC saturated (for GCS traffic), depending
	// on which is the current bottleneck.
	const resolverParallelism = 128
	const visitorParallelism = 128

	// Manufacture a root node.
	fi, err := os.Stat(dir)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	rootNode := &node{
		RelPath: "",
		Info: fs.FileInfo{
			Type:        fs.TypeDirectory,
			Permissions: fi.Mode() & os.ModePerm,
			Uid:         sys.UserId(fi.Sys().(*syscall.Stat_t).Uid),
			Gid:         sys.GroupId(fi.Sys().(*syscall.Stat_t).Gid),
			MTime:       fi.ModTime(),
			Scores:      []blob.Score{score},
		},
	}

	// Create a blob store.
	blobStore := newBlobStore(bucket, objectNamePrefix, crypter)

	// Walk the graph.
	err = dag.Visit(
		ctx,
		[]dag.Node{rootNode},
		newDependencyResolver(blobStore, logger),
		newVisitor(dir, blobStore, logger),
		resolverParallelism,
		visitorParallelism)

	if err != nil {
		err = fmt.Errorf("dag.Visit: %v", err)
		return
	}

	return
}

// newBlobStore creates a blob store that loads blobs from the supplied bucket
// under the given name prefix, decrypting with the supplied crypter.
func newBlobStore(
	bucket gcs.Bucket,
	objectNamePrefix string,
	crypter crypto.Crypter) (bs blob.Store) {
	// Load blobs from GCS.
	bs = blob.NewGCSStore(bucket, objectNamePrefix)

	// Fulfill the contract required by NewGCSStore. This is necessary in tests
	// that write using the blob store created by this function.
	bs = blob.NewExistingScoresStore(util.NewStringSet(), bs)

	// Make paranoid checks on the results.
	bs = blob.NewCheckingStore(bs)

	// Decrypt blob data coming back from GCS.
	bs = blob.NewEncryptingStore(crypter, bs)

	return
}

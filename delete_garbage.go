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

// Delete every object in the bucket with the prefix `garbage/`. (These are
// objects produced by `comeback gc`.)

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
)

var cmdDeleteGarbage = &Command{
	Name: "delete_garbage",
	Run:  runDeleteGarbage,
}

////////////////////////////////////////////////////////////////////////
// Delete garbage
////////////////////////////////////////////////////////////////////////

func runDeleteGarbage(ctx context.Context, args []string) (err error) {
	eg, ctx := errgroup.WithContext(ctx)

	// Grab dependencies.
	bucket := getBucket(ctx)

	// List all garbage objects.
	objects := make(chan *gcs.Object, 100)
	eg.Go(func() (err error) {
		defer close(objects)
		err = gcsutil.ListPrefix(ctx, bucket, garbagePrefix, objects)
		if err != nil {
			err = fmt.Errorf("ListPrefix: %v", err)
			return
		}

		return
	})

	// Count the objects passing through, periodically printing a status update.
	// Convert to names.
	var count uint64
	toDelete := make(chan string)
	eg.Go(func() (err error) {
		defer close(toDelete)
		ticker := time.Tick(2 * time.Second)

		for o := range objects {
			count++

			// Print a status update?
			select {
			case <-ticker:
				log.Printf("%d names seen so far.", count)

			default:
			}

			// Pass on the name.
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return

			case toDelete <- o.Name:
			}
		}

		return
	})

	// Delete the objects.
	eg.Go(func() (err error) {
		err = deleteObjects(ctx, bucket, toDelete)
		if err != nil {
			err = fmt.Errorf("deleteObjects: %v", err)
			return
		}

		return
	})

	err = eg.Wait()
	if err != nil {
		return
	}

	// Print a summary.
	log.Printf("Deleted %d objects.", count)

	return
}

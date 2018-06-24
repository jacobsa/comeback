// create_blobs creates lots of random blobs in a bucket.
//
// This is in an effort to reproduce strange authentication errors from GCS
// when running comeback.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	// Create the bucket.
	bucket, err := createBucket(ctx)
	if err != nil {
		err = fmt.Errorf("creating bucket: %v", err)
		return
	}

	// Create blobs, with parallelism.
	eg, ctx := errgroup.WithContext(ctx)

	const parallelism = 32
	for i := 0; i < parallelism; i++ {
		eg.Go(func() (err error) {
			err = createBlobs(ctx, bucket)
			if err != nil {
				err = fmt.Errorf("creating blobs: %v", err)
				return
			}

			return
		})
	}

	err = eg.Wait()
	return
}

func createBucket(ctx context.Context) (bucket gcs.Bucket, err error) {
	err = errors.New("TODO")
	return
}

// createBlobs repeatedly creates blobs until it fails.
func createBlobs(ctx context.Context, bucket gcs.Bucket) (err error) {
	err = errors.New("TODO")
	return
}

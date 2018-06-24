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
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	err = errors.New("TODO")
	return
}

// create_blobs creates lots of random blobs in a bucket.
//
// This is in an effort to reproduce strange authentication errors from GCS
// when running comeback.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcsutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/sync/errgroup"
)

const (
	keyFile    = "/Users/jacobsa/.comeback.key"
	bucketName = "jacobsa-test"
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
	// Create an oauth2 token source.
	tokenSrc, err := makeTokenSource(ctx)
	if err != nil {
		err = fmt.Errorf("makeTokenSource: %v", err)
		return
	}

	// Turn that into a connection.
	connCfg := &gcs.ConnConfig{
		TokenSource:     tokenSrc,
		MaxBackoffSleep: time.Minute,
	}

	conn, err := gcs.NewConn(connCfg)
	if err != nil {
		err = fmt.Errorf("NewConn: %v", err)
		return
	}

	// Grab the bucket.
	bucket, err = conn.OpenBucket(
		ctx,
		&gcs.OpenBucketOptions{
			Name: bucketName,
		})

	if err != nil {
		err = fmt.Errorf("OpenBucket: %v", err)
		return
	}

	return
}

func makeTokenSource(ctx context.Context) (ts oauth2.TokenSource, err error) {
	// Attempt to read the JSON file.
	contents, err := ioutil.ReadFile(keyFile)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %v", keyFile, err)
		return
	}

	// Create a config struct based on its contents.
	jwtConfig, err := google.JWTConfigFromJSON(contents, gcs.Scope_FullControl)
	if err != nil {
		err = fmt.Errorf("JWTConfigFromJSON: %v", err)
		return
	}

	// Create the token source.
	ts = jwtConfig.TokenSource(ctx)

	return
}

// createBlobs repeatedly creates blobs until it fails.
func createBlobs(ctx context.Context, bucket gcs.Bucket) (err error) {
	for {
		err = createBlob(ctx, bucket)
		if err != nil {
			return
		}
	}
}

// createBlob creates one blob.
func createBlob(ctx context.Context, bucket gcs.Bucket) (err error) {
	// Choose contents.
	contents, err := makeBlobContents()
	if err != nil {
		err = fmt.Errorf("choosing contents: %v", err)
		return
	}

	// Hash them.
	crc32c := *gcsutil.CRC32C(contents)
	md5 := *gcsutil.MD5(contents)

	// Use the hash as the blob name.
	blobName := path.Join("tmp", hex.EncodeToString(md5[:]))

	// Create the blob.
	createReq := &gcs.CreateObjectRequest{
		Name:     blobName,
		Contents: bytes.NewReader(contents),
		CRC32C:   &crc32c,
		MD5:      &md5,
	}

	_, err = bucket.CreateObject(ctx, createReq)
	if err != nil {
		err = fmt.Errorf("CreateObject: %v", err)
		return
	}

	return
}

func makeBlobContents() (b []byte, err error) {
	err = errors.New("TODO")
	return
}

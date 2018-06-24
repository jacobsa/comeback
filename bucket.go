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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/jacobsa/gcloud/gcs"
)

var g_bucketOnce sync.Once
var g_bucket gcs.Bucket

func makeTokenSource(
	ctx context.Context) (ts oauth2.TokenSource, err error) {
	cfg := getConfig()

	// Attempt to read the JSON file.
	contents, err := ioutil.ReadFile(cfg.KeyFile)
	if err != nil {
		err = fmt.Errorf("ReadFile(%q): %v", cfg.KeyFile, err)
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

func makeBucket(ctx context.Context) (bucket gcs.Bucket, err error) {
	cfg := getConfig()

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
			Name: cfg.BucketName,
		})

	if err != nil {
		err = fmt.Errorf("OpenBucket: %v", err)
		return
	}

	return
}

func initBucket(ctx context.Context) {
	var err error

	g_bucket, err = makeBucket(ctx)
	if err != nil {
		panic(err)
	}
}

func getBucket(ctx context.Context) gcs.Bucket {
	g_bucketOnce.Do(func() { initBucket(ctx) })
	return g_bucket
}

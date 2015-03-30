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
	"fmt"
	"sync"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/oauthutil"
	"google.golang.org/cloud/storage"
)

var g_bucketOnce sync.Once
var g_bucket gcs.Bucket

func makeBucket() (bucket gcs.Bucket, err error) {
	cfg := getConfig()

	// Create an authenticated HTTP client.
	httpClient, err := oauthutil.NewJWTHttpClient(
		cfg.KeyFile,
		[]string{storage.ScopeFullControl})

	if err != nil {
		err = fmt.Errorf("NewJWTHttpClient: %v", err)
		return
	}

	// Turn that into a connection.
	connCfg := &gcs.ConnConfig{
		HTTPClient: httpClient,
	}

	conn, err := gcs.NewConn(connCfg)
	if err != nil {
		err = fmt.Errorf("NewConn: %v", err)
		return
	}

	// Grab the bucket.
	bucket = conn.GetBucket(cfg.BucketName)

	return
}

func initBucket() {
	var err error

	g_bucket, err = makeBucket()
	if err != nil {
		panic(err)
	}
}

func getBucket() gcs.Bucket {
	g_bucketOnce.Do(initBucket)
	return g_bucket
}

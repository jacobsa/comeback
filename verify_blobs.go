// Copyright 2013 Aaron Jacobs. All Rights Reserved.
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

// A command that loads blobs in an S3 bucket and makes sure their scores match
// their key names.

package main

import (
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/comeback/blob"
	"log"
)

var cmdVerifyBlobs = &Command{
	Name: "verify_blobs",
}

var g_keyLowerBound =
	cmdVerifyBlobs.Flags.String(
		"key_lb",
		"",
		"The exclusive lower bound for the bucket keys to process.")

var g_keyUpperBound =
	cmdVerifyBlobs.Flags.String(
		"key_ub",
		"",
		"The exclusive upper bound for the bucket keys to process, or the empty " +
		"string if no limit.")

func init() {
	cmdVerifyBlobs.Run = runVerifyBlobs // Break flag-related dependency loop.
}

func reachedUpperBound(key string, upperBound string) bool {
	return upperBound != "" && key >= upperBound
}

func runVerifyBlobs(args []string) {
	cfg := getConfig()
	var err error

	// Open a connection to S3.
	bucket, err := s3.OpenBucket(cfg.S3Bucket, cfg.S3Region, cfg.AccessKey)
	if err != nil {
		log.Fatalln("Creating S3 bucket:", err)
	}

	// Loop over the keys in the bucket.
	for prevKey := *g_keyLowerBound; !reachedUpperBound(prevKey, *g_keyUpperBound); {
		// Grab the next batch of keys.
		keyBatch, err := bucket.ListKeys(prevKey)
		if err != nil {
			log.Fatalln("ListKeys:", err)
		}

		// Are we out of keys?
		if len(keyBatch) == 0 {
			break
		}

		// Process each key.
		for _, key := range keyBatch {
			// Have we gone too far?
			if reachedUpperBound(key, *g_keyUpperBound) {
				break
			}

			// Grab the object.
			data, err := bucket.GetObject(key)
			if err != nil {
				log.Fatalln("GetObject:", err)
			}

			// Check its score.
			hexScore := blob.ComputeScore(data).Hex()
			if hexScore == key {
				log.Printf("OK: %s (%8d bytes)\n", key, len(data))
			} else {
				log.Fatalf("Wrong score for key %s: %s\n", key, hexScore)
			}
		}

		// Move on to the next batch the next time around.
		prevKey = keyBatch[len(keyBatch) - 1]
	}
}

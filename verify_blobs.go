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

var cmdVerifyBlobs = &Command{
	Name: "verify_blobs",
}

var g_keyLowerBound = cmdVerifyBlobs.Flags.String(
	"key_lb",
	"",
	"The exclusive lower bound for the bucket keys to process.")

var g_keyUpperBound = cmdVerifyBlobs.Flags.String(
	"key_ub",
	"",
	"The exclusive upper bound for the bucket keys to process, or the empty "+
		"string if no limit.")

func init() {
	cmdVerifyBlobs.Run = runVerifyBlobs // Break flag-related dependency loop.
}

func reachedUpperBound(key string, upperBound string) bool {
	return upperBound != "" && key >= upperBound
}

func runVerifyBlobs(args []string) {
	panic("Not implemented for GCS.")
}

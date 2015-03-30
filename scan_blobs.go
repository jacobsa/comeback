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

// A command that scans all of the blobs in a GCS bucket, verifying their
// scores against their content.
//
// Output is space-separated lines of the form:
//
//     <score> <child score>*
//
// where <score> is a verified score, and the list is a list of scores of
// children for directories (and empty for files).

package main

var cmdScanBlobs = &Command{
	Name: "scan_blobs",
	Run:  runScanBlobs,
}

func runScanBlobs(args []string) {
	panic("TODO")
}

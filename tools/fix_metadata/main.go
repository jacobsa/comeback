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

// A tool to add missing fields to an existing GCS bucket, for #18.
//
// Input is of the form "<SHA-1> <CRC32C> <MD5>", e.g.:
//
//     e04b25d650dee1dff6ab1743724fa7c184282e94 0x12e9bf88 2bad5bb78f17232ef8c727f59eb82325
//
package main

import (
	"crypto/md5"
	"crypto/sha1"

	"github.com/jacobsa/gcloud/gcs"
	"golang.org/x/net/context"
)

type crc32cChecksum uint32
type md5hash [md5.Size]byte
type sha1hash [sha1.Size]byte

type info struct {
	crc32c crc32cChecksum
	md5    md5hash
}

// List all blob objects in the GCS bucket into the channel.
func listBlobObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	objects chan<- *gcs.Object) (err error)

// Filter to names of objects that lack the appropriate metadata keys.
func filterToProblematicNames(
	ctx context.Context,
	objects <-chan *gcs.Object,
	names chan<- string) (err error)

// For each object name, issue a request to set the appropriate metadata keys
// based on the contents of the supplied map. Write out the names of the
// objects processed.
func fixProblematicObjects(
	ctx context.Context,
	infoMap map[sha1hash]info,
	names <-chan string,
	processed chan<- string) (err error)

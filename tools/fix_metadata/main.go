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
// Input is on stdin, and is of the form "<SHA-1> <CRC32C> <MD5>", e.g.:
//
//     e04b25d650dee1dff6ab1743724fa7c184282e94 0x12e9bf88 2bad5bb78f17232ef8c727f59eb82325
//
package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/gcs/gcstesting"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

type crc32cChecksum uint32
type md5Hash [md5.Size]byte
type sha1Hash [sha1.Size]byte

type checksums struct {
	crc32c crc32cChecksum
	md5    md5Hash
}

// A mapping from SHA-1 to CRC32C and MD5.
type checksumMap map[sha1Hash]checksums

const (
	// Cf. blob_store.go
	blobObjectNamePrefix = "blobs/"

	// Cf. gcs_store.go
	metadataKey_SHA1   = "comeback_sha1"
	metadataKey_CRC32C = "comeback_crc32c"
	metadataKey_MD5    = "comeback_md5"
)

var gInputLineRe = regexp.MustCompile(
	"^([0-9a-f]{40}) (0x[0-9a-f]{8}) ([0-9a-f]{16})$")

func parseInputLine(line []byte) (sha1 sha1Hash, c checksums, err error) {
	// Match against the regexp.
	matches := gInputLineRe.FindSubmatch(line)
	if matches == nil {
		err = errors.New("No match.")
		return
	}

	// Parse each component.
	_, err = hex.Decode(sha1[:], matches[1])
	if err != nil {
		panic(fmt.Sprintf("Unexpected decode error for %q: %v", matches[1], err))
	}

	crc32c64, err := strconv.ParseUint(string(matches[2]), 0, 32)
	if err != nil {
		panic(fmt.Sprintf("Unexpected decode error for %q: %v", matches[2], err))
	}

	c.crc32c = crc32cChecksum(crc32c64)

	_, err = hex.Decode(c.md5[:], matches[3])
	if err != nil {
		panic(fmt.Sprintf("Unexpected decode error for %q: %v", matches[3], err))
	}

	return
}

// Read the supplied input file, producing a checksum map.
func parseInput(in io.Reader) (m checksumMap, err error) {
	m = make(checksumMap)

	// Scan each line.
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		var sha1 sha1Hash
		var c checksums
		sha1, c, err = parseInputLine(scanner.Bytes())
		if err != nil {
			err = fmt.Errorf("Parsing input line %q: %v", scanner.Text(), err)
			return
		}

		m[sha1] = c
	}

	// Was there an error scanning?
	if scanner.Err() != nil {
		err = fmt.Errorf("Scanning: %v", scanner.Err())
		return
	}

	return
}

// List all blob objects in the GCS bucket into the channel.
func listBlobObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	objects chan<- *gcs.Object) (err error) {
	req := &gcs.ListObjectsRequest{
		Prefix: blobObjectNamePrefix,
	}

	// List until we run out.
	for {
		// Fetch the next batch.
		var listing *gcs.Listing
		listing, err = bucket.ListObjects(ctx, req)
		if err != nil {
			err = fmt.Errorf("ListObjects: %v", err)
			return
		}

		// Pass on each object.
		for _, o := range listing.Objects {
			// Special case: for gcsfuse compatibility, we allow blobObjectNamePrefix
			// to exist as its own object name. Skip it.
			if o.Name == blobObjectNamePrefix {
				continue
			}

			select {
			case objects <- o:

				// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}
		}

		// Are we done?
		if listing.ContinuationToken == "" {
			break
		}

		req.ContinuationToken = listing.ContinuationToken
	}

	return
}

// Filter to names of objects that lack the appropriate metadata keys.
func filterToProblematicNames(
	ctx context.Context,
	objects <-chan *gcs.Object,
	names chan<- string) (err error) {
	for o := range objects {
		// Skip objects that already have all of the keys.
		_, ok0 := o.Metadata[metadataKey_SHA1]
		_, ok1 := o.Metadata[metadataKey_CRC32C]
		_, ok2 := o.Metadata[metadataKey_MD5]

		if ok0 && ok1 && ok2 {
			continue
		}

		// Pass on the names of others.
		select {
		case names <- o.Name:

			// Cancelled?
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

// For each object name, issue a request to set the appropriate metadata keys
// based on the contents of the supplied map. Write out the names of the
// objects processed, and those for whom info wasn't available.
func fixProblematicObjects(
	ctx context.Context,
	info checksumMap,
	names <-chan string,
	processed chan<- string,
	unknown chan<- string) (err error)

func run(
	bucket gcs.Bucket,
	info checksumMap) (err error) {
	b := syncutil.NewBundle(context.Background())
	defer func() { err = b.Join() }()

	// List all of the blob objects.
	objectRecords := make(chan *gcs.Object, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(objectRecords)
		err = listBlobObjects(ctx, bucket, objectRecords)
		if err != nil {
			err = fmt.Errorf("listBlobObjects: %v", err)
			return
		}

		return
	})

	// Filter to the ones we need to fix up.
	problematicNames := make(chan string, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(problematicNames)
		err = filterToProblematicNames(ctx, objectRecords, problematicNames)
		if err != nil {
			err = fmt.Errorf("filterToProblematicNames: %v", err)
			return
		}

		return
	})

	// Fix those objects with some parallelism.
	const parallelism = 128
	panic("TODO")

	// Log status updates, and at the end log the objects that were not
	// processed, returning an error if non-zero.
	panic("TODO")

	return
}

func panicIf(err *error) {
	if *err != nil {
		panic(*err)
	}
}

func main() {
	flag.Parse()

	// Panic if anything below fails.
	var err error
	defer panicIf(&err)

	// Parse the input.
	info, err := parseInput(os.Stdin)
	if err != nil {
		err = fmt.Errorf("parseInput: %v", err)
		return
	}

	// Run.
	err = run(gcstesting.IntegrationTestBucketOrDie(), info)
	if err != nil {
		err = fmt.Errorf("run: %v", err)
		return
	}

	panic("TODO")
}

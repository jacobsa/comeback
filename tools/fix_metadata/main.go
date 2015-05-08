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
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/oauthutil"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var fKeyFile = flag.String("key_file", "", "")
var fBucket = flag.String("bucket", "", "")

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
	"^([0-9a-f]{40}) (0x[0-9a-f]{8}) ([0-9a-f]{32})$")

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

// Parse the object name into its expected SHA-1 hash.
func parseObjectName(name string) (sha1 sha1Hash, err error) {
	if !strings.HasPrefix(name, blobObjectNamePrefix) {
		err = fmt.Errorf("Expected prefix")
		return
	}

	hexSha1 := strings.TrimPrefix(name, blobObjectNamePrefix)
	score, err := blob.ParseHexScore(hexSha1)
	if err != nil {
		err = fmt.Errorf("ParseHexScore: %v", err)
		return
	}

	sha1 = sha1Hash(score)
	return
}

// For each object name, issue a request to set the appropriate metadata keys
// based on the contents of the supplied map. Write out the names of the
// objects processed, and those for whom info wasn't available.
func fixProblematicObjects(
	ctx context.Context,
	bucket gcs.Bucket,
	info checksumMap,
	names <-chan string,
	processed chan<- string,
	unknown chan<- string) (err error) {
	for name := range names {
		// Parse the name.
		var sha1 sha1Hash
		sha1, err = parseObjectName(name)
		if err != nil {
			err = fmt.Errorf("Parsing object name %q: %v", name, err)
			return
		}

		// Do we have info for this object?
		c, ok := info[sha1]
		if !ok {
			select {
			case unknown <- name:

				// Cancelled?
			case <-ctx.Done():
				err = ctx.Err()
				return
			}

			continue
		}

		// Fix it up.
		//
		// Formats cf. gcs_store.go.
		sha1Str := hex.EncodeToString(sha1[:])
		crc32cStr := fmt.Sprintf("%#08x", c.crc32c)
		md5Str := hex.EncodeToString(c.md5[:])

		req := &gcs.UpdateObjectRequest{
			Name: name,
			Metadata: map[string]*string{
				metadataKey_SHA1:   &sha1Str,
				metadataKey_CRC32C: &crc32cStr,
				metadataKey_MD5:    &md5Str,
			},
		}

		_, err = bucket.UpdateObject(ctx, req)
		if err != nil {
			err = fmt.Errorf("UpdateObject: %v", err)
			return
		}

		// Pass on the name as processed.
		select {
		case processed <- name:

			// Cancelled?
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
	}

	return
}

// Log status updates, and at the end log the objects that were not
// processed, returning an error if non-zero.
func monitorProgress(
	ctx context.Context,
	processedChan <-chan string,
	unknownChan <-chan string) (err error) {
	var processed int
	var unknown int

	// Set up a ticker for logging status updates.
	const period = time.Second
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	// Keep going until both channels are closed.
	for processedChan != nil || unknownChan != nil {
		select {
		case <-ticker.C:
			log.Printf("%v processed successfully, %v unknown", processed, unknown)

		case _, ok := <-processedChan:
			if ok {
				processed++
			} else {
				processedChan = nil
			}

		case name, ok := <-unknownChan:
			if ok {
				log.Printf("Unknown object: %q", name)
				unknown++
			} else {
				unknownChan = nil
			}
		}
	}

	// Return an error if any object was unknown.
	if unknown != 0 {
		err = fmt.Errorf("%v unknown objects", unknown)
		return
	}

	return
}

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
	var wg sync.WaitGroup

	processed := make(chan string, 100)
	unknown := make(chan string, 100)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		b.Add(func(ctx context.Context) (err error) {
			defer wg.Done()
			err = fixProblematicObjects(
				ctx,
				bucket,
				info,
				problematicNames,
				processed,
				unknown)

			if err != nil {
				err = fmt.Errorf("fixProblematicObjects: %v", err)
				return
			}

			return
		})
	}

	go func() {
		wg.Wait()
		close(processed)
		close(unknown)
	}()

	// Log status updates, and at the end log the objects that were not
	// processed, returning an error if non-zero.
	b.Add(func(ctx context.Context) (err error) {
		err = monitorProgress(ctx, processed, unknown)
		if err != nil {
			err = fmt.Errorf("monitorProgress: %v", err)
			return
		}

		return
	})

	return
}

func panicIf(err *error) {
	if *err != nil {
		panic(*err)
	}
}

func getHTTPClient() (client *http.Client, err error) {
	if *fKeyFile == "" {
		err = errors.New("You must set --key_file.")
		return
	}

	const scope = gcs.Scope_FullControl
	client, err = oauthutil.NewJWTHttpClient(*fKeyFile, []string{scope})
	if err != nil {
		err = fmt.Errorf("oauthutil.NewJWTHttpClient: %v", err)
		return
	}

	return
}

func bucketName() (name string, err error) {
	name = *fBucket
	if name == "" {
		err = errors.New("You must set --bucket.")
		return
	}

	return
}

func getBucket() (b gcs.Bucket) {
	var err error
	defer panicIf(&err)

	// Get the HTTP client,
	client, err := getHTTPClient()
	if err != nil {
		err = fmt.Errorf("IntegrationTestHTTPClient: %v", err)
		return
	}

	// Find the bucket name.
	name, err := bucketName()
	if err != nil {
		err = fmt.Errorf("bucketName: %v", err)
		return
	}

	// Create a connection.
	cfg := &gcs.ConnConfig{
		HTTPClient:      client,
		MaxBackoffSleep: 30 * time.Second,
	}

	conn, err := gcs.NewConn(cfg)
	if err != nil {
		err = fmt.Errorf("gcs.NewConn: %v", err)
		return
	}

	// Extract the bucket.
	b = conn.GetBucket(name)

	return
}

func main() {
	flag.Parse()
	bucket := getBucket()
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Panic if anything below fails.
	var err error
	defer func() {
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Parse the input.
	log.Println("Parsing input...")
	info, err := parseInput(os.Stdin)
	if err != nil {
		err = fmt.Errorf("parseInput: %v", err)
		return
	}

	log.Println("Done parsing input.")

	// Run.
	err = run(bucket, info)
	if err != nil {
		err = fmt.Errorf("run: %v", err)
		return
	}
}

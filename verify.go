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

// A command that reads all blobs necessary for verifying the directory
// structure rooted at a set of backup root scores, ensuring that the entire
// directory structure is intact in GCS.
//
// Optionally, all file content is also read and verified. This is less
// important than verifying directory connectedness if we trust that GCS does
// not corrupt object metadata (where we store expected CRC32C and MD5) and
// does correctly report the object's CRC32C and MD5 sums in listings,
// verifying them periodically.
//
// Output of the following form is written to stdout and a file in the user's
// home directory:
//
//     <timestamp> <node> [<child node> ...]
//
// where:
//
//  *  Timestamps are formatted according to time.RFC3339.
//
//  *  Node names have one of two forms:
//
//     *   Nodes of the form "d:<hex score>" represent the directory listing
//         contained within the blob of the given score.
//
//     *   Nodes of the form "f:<hex score>" represent a piece of a file,
//         contained within the blob of the given score.
//
// An output line for a directory node means that at the given timestamp we
// certified that a piece of content with the given score was parseable as a
// directory listing that referred to the given scores for its direct children.
//
// An output line for a file node means that at the given timestamp we
// certified that a piece of content with the given score was parseable as a
// piece of a file. File nodes never have children.

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/util"
	"github.com/jacobsa/comeback/verify"
	"github.com/jacobsa/comeback/wiring"
	"github.com/jacobsa/gcloud/gcs"
	"github.com/jacobsa/gcloud/syncutil"
	"golang.org/x/net/context"
)

var cmdVerify = &Command{
	Name: "verify",
}

// TODO(jacobsa): Get these automatically from the registry.
var fRoots = cmdVerify.Flags.String(
	"roots",
	"",
	"Comma-separated list of backup root scores to verify.")

var fFast = cmdVerify.Flags.Bool(
	"fast",
	false,
	"When set, don't verify file content.")

func init() {
	cmdVerify.Run = runVerify // Break flag-related dependency loop.
}

////////////////////////////////////////////////////////////////////////
// Visitor types
////////////////////////////////////////////////////////////////////////

// A record certifying something we confirmed about a node at a certain time.
// See the notes at the top of verify.go for details.
type verifyRecord struct {
	t        time.Time
	node     verify.Node
	adjacent []verify.Node
}

// A visitor that writes the information it gleans from the wrapped visitor to
// a channel.
type snoopingVisitor struct {
	records chan<- verifyRecord
	wrapped graph.Visitor
}

func (v *snoopingVisitor) Visit(
	ctx context.Context,
	node string) (adjacent []string, err error) {
	// Call through.
	adjacent, err = v.wrapped.Visit(ctx, node)
	if err != nil {
		return
	}

	// Build a record.
	r := verifyRecord{
		t: time.Now(),
	}

	r.node, err = verify.ParseNode(node)
	if err != nil {
		err = fmt.Errorf("ParseNode(%q): %v", node, err)
		return
	}

	for _, a := range adjacent {
		var adjacentNode verify.Node
		adjacentNode, err = verify.ParseNode(a)
		if err != nil {
			err = fmt.Errorf("ParseNode(%q): %v", a, err)
			return
		}

		r.adjacent = append(r.adjacent, adjacentNode)
	}

	// Write out the record.
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return

	case v.records <- r:
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Output
////////////////////////////////////////////////////////////////////////

// Print output based on the visitor results arriving on the supplied channel.
func formatVerifyOutput(r verifyRecord) (s string) {
	s = fmt.Sprintf(
		"%s %s",
		r.t.Format(time.RFC3339),
		r.node)

	for _, a := range r.adjacent {
		s += fmt.Sprintf(" %s", a.String())
	}

	return
}

// Parse the supplied line (without line break) previously output by the verify
// command.
func parseVerifyRecord(line []byte) (r verifyRecord, err error) {
	// We expect space-separate components.
	components := bytes.Split(line, []byte{' '})
	if len(components) < 2 {
		err = fmt.Errorf(
			"Expected at least two components, got %d.",
			len(components))

		return
	}

	// The first should be the timestmap.
	r.t, err = time.Parse(time.RFC3339, string(components[0]))
	if err != nil {
		err = fmt.Errorf("time.Parse(%q): %v", components[0], err)
		return
	}

	// The rest are node names.
	var nodes []verify.Node
	for i := 1; i < len(components); i++ {
		c := components[i]

		var node verify.Node
		node, err = verify.ParseNode(string(c))
		if err != nil {
			err = fmt.Errorf("ParseNode(%q): %v", c, err)
			return
		}

		nodes = append(nodes, node)
	}

	// Apportion nodes.
	r.node = nodes[0]
	r.adjacent = nodes[1:]

	return
}

// Parse the supplied output from a run of the verify command, writing records
// into the supplied channel.
func parseVerifyOutput(
	ctx context.Context,
	in io.Reader,
	records chan<- verifyRecord) (err error) {
	reader := bufio.NewReader(in)

	for {
		// Find the next line. EOF with no data means we are done; otherwise ignore
		// EOF in case the file doesn't end with a newline.
		var line []byte
		line, err = reader.ReadBytes('\n')
		if err == io.EOF {
			err = nil
			if len(line) == 0 {
				break
			}
		}

		// Propagate other errors.
		if err != nil {
			err = fmt.Errorf("ReadBytes: %v", err)
			return
		}

		// Trim the delimiter.
		line = line[:len(line)-1]

		// Parse the line.
		var r verifyRecord
		r, err = parseVerifyRecord(line)
		if err != nil {
			err = fmt.Errorf("parseVerifyRecord(%q): %v", line, err)
			return
		}

		// Write out the record.
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return

		case records <- r:
		}
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// List blob.ListScores, but returns a slice instead of writing into a channel.
func listAllScores(
	ctx context.Context,
	bucket gcs.Bucket,
	namePrefix string) (scores []blob.Score, err error) {
	b := syncutil.NewBundle(context.Background())
	defer func() { err = b.Join() }()

	// List scores into a channel.
	scoreChan := make(chan blob.Score, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(scoreChan)
		err = blob.ListScores(ctx, bucket, wiring.BlobObjectNamePrefix, scoreChan)
		if err != nil {
			err = fmt.Errorf("ListScores: %v", err)
			return
		}

		return
	})

	// Accumulate into the slice.
	b.Add(func(ctx context.Context) (err error) {
		for score := range scoreChan {
			scores = append(scores, score)
		}

		return
	})

	return
}

////////////////////////////////////////////////////////////////////////
// Verify
////////////////////////////////////////////////////////////////////////

// Run the verification pipeline. Return a count of the number of scores
// verified and the number skipped due to readFiles being false.
func verifyImpl(
	ctx context.Context,
	readFiles bool,
	rootScores []blob.Score,
	knownScores []blob.Score,
	blobStore blob.Store,
	output io.Writer) (nodesVerified uint64, nodesSkipped uint64, err error) {
	b := syncutil.NewBundle(ctx)

	// Visit every node in the graph, snooping on the graph structure into a
	// channel.
	visitorRecords := make(chan verifyRecord, 100)
	b.Add(func(ctx context.Context) (err error) {
		defer close(visitorRecords)

		visitor := verify.NewVisitor(
			readFiles,
			knownScores,
			blobStore)

		visitor = &snoopingVisitor{
			wrapped: visitor,
			records: visitorRecords,
		}

		// Format root node names.
		var roots []string
		for _, score := range rootScores {
			roots = append(roots, verify.FormatNodeName(true, score))
		}

		// Traverse starting at the specified roots. Use an "experimentally
		// determined" parallelism, which in theory should depend on bandwidth-delay
		// products but in practice comes down to when the OS gets cranky about open
		// files.
		const parallelism = 128

		err = graph.Traverse(
			context.Background(),
			parallelism,
			roots,
			visitor)

		if err != nil {
			err = fmt.Errorf("Traverse: %v", err)
			return
		}

		return
	})

	// Count and output the nodes visited, filtering out file nodes if we're not
	// actually reading and verifying them.
	b.Add(func(ctx context.Context) (err error) {
		for r := range visitorRecords {
			var dir bool
			dir, _, err = verify.ParseNodeName(r.node)
			if err != nil {
				err = fmt.Errorf("ParseNodeName(%q): %v", r.node, err)
				return
			}

			// Skip files if appropriate.
			if !readFiles && !dir {
				nodesSkipped++
				continue
			}

			// Increment the count.
			nodesVerified++

			// Output the information.
			_, err = fmt.Fprintf(output, "%s\n", formatVerifyOutput(r))
			if err != nil {
				err = fmt.Errorf("Fprintf: %v", err)
				return
			}
		}

		return
	})

	err = b.Join()
	return
}

// Open the file to which we log verify output.
func openVerifyLog() (w io.WriteCloser, err error) {
	// Find the current user.
	u, err := user.Current()
	if err != nil {
		err = fmt.Errorf("user.Current: %v", err)
		return
	}

	// Put the file in her home directory. Append to whatever is already there.
	w, err = os.OpenFile(
		path.Join(u.HomeDir, ".comeback.verify.log"),
		os.O_WRONLY|os.O_APPEND|os.O_CREATE,
		0600)

	if err != nil {
		err = fmt.Errorf("OpenFile: %v", err)
		return
	}

	return
}

func runVerify(args []string) {
	// Allow parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Die on error.
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()

	// Read flags.
	readFiles := !*fFast

	if *fRoots == "" {
		err = fmt.Errorf("You must set --roots.")
		return
	}

	rootHexScores := strings.Split(*fRoots, ",")
	var rootScores []blob.Score
	for _, hexScore := range rootHexScores {
		var score blob.Score
		score, err = blob.ParseHexScore(hexScore)
		if err != nil {
			err = fmt.Errorf("Invalid root %q: %v", hexScore, err)
			return
		}

		rootScores = append(rootScores, score)
	}

	// Grab dependencies.
	bucket := getBucket()
	crypter := getCrypter()

	// Open the log file.
	logFile, err := openVerifyLog()
	if err != nil {
		err = fmt.Errorf("openVerifyLog: %v", err)
		return
	}

	defer logFile.Close()

	// Create a blob store.
	blobStore, err := wiring.MakeBlobStore(
		bucket,
		crypter,
		util.NewStringSet())

	if err != nil {
		err = fmt.Errorf("MakeBlobStore: %v", err)
		return
	}

	// List all scores in the bucket, verifying the object record metadata in the
	// process.
	log.Println("Listing scores...")
	knownScores, err := listAllScores(
		context.Background(),
		bucket,
		wiring.BlobObjectNamePrefix)

	if err != nil {
		err = fmt.Errorf("listAllScores: %v", err)
		return
	}

	log.Printf("Listed %d scores.", len(knownScores))

	// Run the rest of the pipeline.
	nodesVerified, nodesSkipped, err := verifyImpl(
		context.Background(),
		readFiles,
		rootScores,
		knownScores,
		blobStore,
		io.MultiWriter(logFile, os.Stdout))

	if err != nil {
		err = fmt.Errorf("verifyImpl: %v", err)
		return
	}

	log.Printf(
		"Successfully verified %d nodes (%d skipped due to fast mode).",
		nodesVerified,
		nodesSkipped)

	return
}

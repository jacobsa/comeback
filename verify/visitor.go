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

package verify

import (
	"fmt"
	"strings"

	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/comeback/repr"
	"golang.org/x/net/context"
)

const (
	filePrefix = "f:"
	dirPrefix  = "d:"
)

// Create a visitor for the DAG of blobs in the supplied bucket. Node names are
// expected to be generated by FormatNodeName.
//
// The visitor reads directory blobs, parses them, and emits their children as
// adjacent nodes. For file nodes, the visitor verifies that their score exists
// (according to allScores), and verifies that the blob can be loaded if
// readFiles is true.
//
// It is expected that the blob store's Load method does score verification for
// us.
func NewVisitor(
	readFiles bool,
	allScores []blob.Score,
	bs blob.Store) (v graph.Visitor) {
	typed := &visitor{
		readFiles:   readFiles,
		blobStore:   bs,
		knownScores: make(map[blob.Score]struct{}),
	}

	for _, score := range allScores {
		typed.knownScores[score] = struct{}{}
	}

	v = typed
	return
}

// Create a node name that can be consumed by the visitor and by ParseNodeName.
// If dir is false, the node represents a file.
func FormatNodeName(
	dir bool,
	score blob.Score) (node string) {
	if dir {
		node = dirPrefix + score.Hex()
	} else {
		node = filePrefix + score.Hex()
	}

	return
}

// Parse a node name created by FormatNodeName.
func ParseNodeName(
	node string) (dir bool, score blob.Score, err error) {
	var hexScore string

	switch {
	case strings.HasPrefix(node, filePrefix):
		hexScore = strings.TrimPrefix(node, filePrefix)

	case strings.HasPrefix(node, dirPrefix):
		dir = true
		hexScore = strings.TrimPrefix(node, dirPrefix)

	default:
		err = fmt.Errorf("Unknown prefix")
		return
	}

	score, err = blob.ParseHexScore(hexScore)
	if err != nil {
		err = fmt.Errorf("ParseHexScore: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Implementation
////////////////////////////////////////////////////////////////////////

type visitor struct {
	readFiles   bool
	blobStore   blob.Store
	knownScores map[blob.Score]struct{}
}

func (v *visitor) visitFile(
	ctx context.Context,
	score blob.Score) (err error) {
	// If reading files is disabled, simply check that the score is known.
	if !v.readFiles {
		_, ok := v.knownScores[score]
		if !ok {
			err = fmt.Errorf("Unknown file score: %s", score.Hex())
			return
		}

		return
	}

	// Make sure we can load the blob contents. Presumably the blob store
	// verifies the score (of the ciphertext) on the way through.
	_, err = v.blobStore.Load(score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", score.Hex(), err)
		return
	}

	return
}

func (v *visitor) visitDir(
	ctx context.Context,
	score blob.Score) (adjacent []string, err error) {
	// Load the blob contents.
	contents, err := v.blobStore.Load(score)
	if err != nil {
		err = fmt.Errorf("Load(%s): %v", score.Hex(), err)
		return
	}

	// Parse the listing.
	listing, err := repr.UnmarshalDir(contents)
	if err != nil {
		err = fmt.Errorf("UnmarshalDir(%s): %v", score.Hex(), err)
		return
	}

	// Return a node for each score in each entry.
	for _, entry := range listing {
		// Is this a directory?
		var dir bool
		switch entry.Type {
		case fs.TypeFile:
			dir = false

		case fs.TypeDirectory:
			dir = true

		case fs.TypeSymlink:
			if len(entry.Scores) != 0 {
				err = fmt.Errorf(
					"Dir %s: symlink unexpectedly contains scores",
					score.Hex())

				return
			}

		default:
			err = fmt.Errorf("Dir %s: unknown entry type %v", score.Hex(), entry.Type)
			return
		}

		// Return a node for each score.
		for _, score := range entry.Scores {
			adjacent = append(adjacent, FormatNodeName(dir, score))
		}
	}

	return
}

func (v *visitor) Visit(
	ctx context.Context,
	node string) (adjacent []string, err error) {
	// Parse the node name.
	dir, score, err := ParseNodeName(node)
	if err != nil {
		err = fmt.Errorf("ParseNodeName(%q): %v", node, err)
		return
	}

	if dir {
		adjacent, err = v.visitDir(ctx, score)
		return
	} else {
		err = v.visitFile(ctx, score)
		return
	}
}

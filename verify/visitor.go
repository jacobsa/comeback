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
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/graph"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a visitor for the DAG of blobs in the supplied bucket. Node names are
// expected to be of the form
//
//  *  "d:<hex score>" for directories, or
//  *  "f:<hex score>" for files.
//
// The visitor reads directory blobs, verifies their score, parses them, and
// emits their children as adjacent nodes. For file nodes, the visitor verifies
// that their score exists (according to allScores), and reads and verifies
// their score if readFiles is true.
func NewVisitor(
	readFiles bool,
	allScores []blob.Score,
	bucket gcs.Bucket,
	blobObjectNamePrefix string) (v graph.Visitor)

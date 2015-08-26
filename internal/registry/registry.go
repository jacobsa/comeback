// Copyright 2012 Aaron Jacobs. All Rights Reserved.
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

package registry

import (
	"time"

	"golang.org/x/net/context"

	"github.com/jacobsa/comeback/internal/blob"
)

type Registry interface {
	// Record that the named backup job has completed.
	RecordBackup(ctx context.Context, j CompletedJob) (err error)

	// Return a list of all completed backups.
	ListBackups(ctx context.Context) (jobs []CompletedJob, err error)

	// Find a particular completed job by start time.
	FindBackup(
		ctx context.Context,
		startTime time.Time) (job CompletedJob, err error)
}

// A record in the backup registry describing a successful backup job.
type CompletedJob struct {
	// The time at which the backup was started.
	StartTime time.Time

	// The name of the backup job.
	Name string

	// The score representing the contents of the backup.
	Score blob.Score
}

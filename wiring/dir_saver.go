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

package wiring

import (
	"github.com/jacobsa/comeback/backup"
	"github.com/jacobsa/gcloud/gcs"
)

// Create a directory saver that stores blobs in the supplied bucket,
// encrypting with a key derived from the given password. If the bucket has
// been used by comeback in the past, the password must match the password used
// previously or an error will be returned.
func MakeDirSaver(
	password string,
	bucket gcs.Bucket) (ds backup.DirectorySaver, err error)

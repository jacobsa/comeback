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

package config

import (
	"github.com/jacobsa/aws"
	"github.com/jacobsa/aws/s3"
	"github.com/jacobsa/aws/sdb"
	"regexp"
)

type Job struct {
	// The path on the file system that should be backed up.
	BasePath string

	// A set of regexps to be matched against relative paths within the base
	// path, excluding the base path itself. If a relative path matches any of
	// these, it will be excluded from the backup. If the path represents a
	// directory, its contents will also be excluded.
	Excludes []*regexp.Regexp
}

type Config struct {
	// A set of named jobs. The names must be valid UTF-8.
	Jobs map[string]*Job

	// The key to be used for signing requests to S3 and SimpleDB.
	AccessKey aws.AccessKey

	// The name of the S3 bucket in which to store blobs, and its associated
	// region.
	S3Bucket string
	S3Region s3.Region

	// The name of the SimpleDB domain in which to store backup history, and its
	// associated region.
	SdbDomain string
	SdbRegion sdb.Region
}

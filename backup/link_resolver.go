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

package backup

import ()

// A stateful object that knows how to keep track of files that are hard-linked
// together.
type LinkResolver interface {
	// Register the supplied path, which points to the given inode on the given
	// containing device. Return a path that has already been registered here, if
	// any.
	Register(containingDevice int32, inode uint64, path string) *string
}

// Create an empty link resolver.
func newLinkResolver() LinkResolver {
	// TODO
	return nil
}

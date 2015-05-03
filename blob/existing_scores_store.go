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

package blob

// Create a blob store that wraps another, responding to calls as follows:
//
//  *  Flush will panic; it is assumed that any buffering and early returning
//     is in front of this store, and that the wrapped store responds to Store
//     only when the blob is durable.
//
//  *  Contains will be responded to directly by this store based on the
//     contents of existingScores. It is assumed that existingScores initially
//     contains only scores that are durable in the wrapped store.
//
//  *  Store will be forwarded to the wrapped store. When it succeeds,
//     existingScores will be updated.
//
//  *  Load will be forwarded to the wrapped store.
//
func NewExistingScoresStore(
	existingScores StringSet,
	wrapped Store) (store Store)

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

// Package repr contains functions useful for serializing and deserializing
// directory listing structs.
package repr

import (
	"code.google.com/p/goprotobuf/proto"
	"github.com/jacobsa/comeback/fs"
)

func convertEntryProto(entryProto *DirectoryEntryProto) (*fs.DirectoryEntry, error) {
	entry := &fs.DirectoryEntry{}

	if entryProto.Name != nil { entry.Name = *entryProto.Name }
	if entryProto.Permissions != nil { entry.Permissions = *entryProto.Permissions }

	return entry, nil
}

// Unmarshal recovers a list of directory entries from bytes previously
// returned by Marshal.
func Unmarshal(d []byte) (entries []*fs.DirectoryEntry, err error) {
	// Parse the protocol buffer.
	listingProto := new(DirectoryListingProto)
	err = proto.Unmarshal(d, listingProto)
	if err != nil {
		return
	}

	// Convert each entry.
	entries = []*fs.DirectoryEntry{}
	for _, entryProto := range listingProto.Entry {
		entry, err := convertEntryProto(entryProto)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return
}

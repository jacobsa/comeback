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
	"fmt"
	"github.com/jacobsa/comeback/fs"
)

func convertProtoType(t DirectoryEntryProto_Type) (fs.EntryType, error) {
	switch t {
	case DirectoryEntryProto_TYPE_FILE:
		return fs.TypeFile, nil
	case DirectoryEntryProto_TYPE_DIRECTORY:
		return fs.TypeDirectory, nil
	case DirectoryEntryProto_TYPE_SYMLINK:
		return fs.TypeSymlink, nil
	}

	return 0, fmt.Errorf("Unrecognized DirectoryEntryProto_Type: %v", t)
}

func convertEntryProto(entryProto *DirectoryEntryProto) (entry *fs.DirectoryEntry, err error) {
	entry = &fs.DirectoryEntry{}

	entry.Name = entryProto.GetName()
	entry.Permissions = entryProto.GetPermissions()

	// Attempt to convert the type.
	entry.Type, err = convertProtoType(entryProto.GetType())
	if err != nil {
		return nil, err
	}

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
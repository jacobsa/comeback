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

func convertType(t fs.EntryType) (DirectoryEntryProto_Type, error) {
	switch t {
	case fs.TypeFile:
		return DirectoryEntryProto_TYPE_FILE, nil
	case fs.TypeDirectory:
		return DirectoryEntryProto_TYPE_DIRECTORY, nil
	case fs.TypeSymlink:
		return DirectoryEntryProto_TYPE_SYMLINK, nil
	}

	return 0, fmt.Errorf("Unrecognized EntryType: %v", t)
}

func makeEntryProto(entry *fs.DirectoryEntry) (*DirectoryEntryProto, error) {
	blobs := []*BlobInfoProto{}
	for _, score := range entry.Scores {
		proto := &BlobInfoProto{Hash: score.Sha1Hash()}
		blobs = append(blobs, proto)
	}

	entryProto := &DirectoryEntryProto{
		Permissions: proto.Uint32(entry.Permissions),
		Name:        proto.String(entry.Name),
		Mtime: &TimeProto{
			Second:     proto.Int64(entry.MTime.Unix()),
			Nanosecond: proto.Uint32(uint32(entry.MTime.Nanosecond())),
		},
		Blob: blobs,
	}

	// Convert the entry's type.
	typeEnum, err := convertType(entry.Type)
	if err != nil {
		return nil, err
	}

	entryProto.Type = typeEnum.Enum()

	return entryProto, nil
}

// Marshal turns a list of directory entries into bytes that can later be used
// with Unmarshal.
func Marshal(entries []*fs.DirectoryEntry) (d []byte, err error) {
	entryProtos := []*DirectoryEntryProto{}
	for _, entry := range entries {
		entryProto, err := makeEntryProto(entry)
		if err != nil {
			return nil, err
		}

		entryProtos = append(entryProtos, entryProto)
	}

	listingProto := &DirectoryListingProto{Entry: entryProtos}
	return proto.Marshal(listingProto)
}

// Unmarshal recovers a list of directory entries from bytes previously
// returned by Marshal.
func Unmarshal(d []byte) (entries []*fs.DirectoryEntry, err error) {
	return nil, fmt.Errorf("TODO")
}

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

package repr

import (
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr/proto"
)

func convertType(t fs.EntryType) (repr_proto.DirectoryEntryProto_Type, error) {
	switch t {
	case fs.TypeFile:
		return repr_proto.DirectoryEntryProto_TYPE_FILE, nil
	case fs.TypeDirectory:
		return repr_proto.DirectoryEntryProto_TYPE_DIRECTORY, nil
	case fs.TypeSymlink:
		return repr_proto.DirectoryEntryProto_TYPE_SYMLINK, nil
	case fs.TypeBlockDevice:
		return repr_proto.DirectoryEntryProto_TYPE_BLOCK_DEVICE, nil
	case fs.TypeCharDevice:
		return repr_proto.DirectoryEntryProto_TYPE_CHAR_DEVICE, nil
	case fs.TypeNamedPipe:
		return repr_proto.DirectoryEntryProto_TYPE_NAMED_PIPE, nil
	}

	return 0, fmt.Errorf("Unrecognized EntryType: %v", t)
}

func makeEntryProto(entry *fs.DirectoryEntry) (*repr_proto.DirectoryEntryProto, error) {
	blobs := []*repr_proto.BlobInfoProto{}
	for _, score := range entry.Scores {
		proto := &repr_proto.BlobInfoProto{Hash: score.Sha1Hash()}
		blobs = append(blobs, proto)
	}

	entryProto := &repr_proto.DirectoryEntryProto{
		Permissions: proto.Uint32(uint32(entry.Permissions)),
		Uid:         proto.Uint32(uint32(entry.Uid)),
		Username:    entry.Username,
		Gid:         proto.Uint32(uint32(entry.Gid)),
		Groupname:   entry.Groupname,
		HardLinkTarget:    entry.HardLinkTarget,
		Name:        proto.String(entry.Name),
		Mtime: &repr_proto.TimeProto{
			Second:     proto.Int64(entry.MTime.Unix()),
			Nanosecond: proto.Uint32(uint32(entry.MTime.Nanosecond())),
		},
		Blob: blobs,
	}

	// Handle symlink targets.
	if entry.Type == fs.TypeSymlink {
		entryProto.Target = proto.String(entry.Target)
	}

	// Handle device numbers.
	if entry.Type == fs.TypeBlockDevice || entry.Type == fs.TypeCharDevice {
		entryProto.DeviceNumber = proto.Int32(entry.DeviceNumber)
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
// with Unmarshal. Note that ContainingDevice and Inode fields are lost.
func Marshal(entries []*fs.DirectoryEntry) (d []byte, err error) {
	entryProtos := []*repr_proto.DirectoryEntryProto{}
	for _, entry := range entries {
		entryProto, err := makeEntryProto(entry)
		if err != nil {
			return nil, err
		}

		entryProtos = append(entryProtos, entryProto)
	}

	listingProto := &repr_proto.DirectoryListingProto{Entry: entryProtos}
	return proto.Marshal(listingProto)
}

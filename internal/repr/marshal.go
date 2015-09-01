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
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr/proto"
)

func convertType(t fs.Type) (repr_proto.FileInfoProto_Type, error) {
	switch t {
	case fs.TypeFile:
		return repr_proto.FileInfoProto_TYPE_FILE, nil
	case fs.TypeDirectory:
		return repr_proto.FileInfoProto_TYPE_DIRECTORY, nil
	case fs.TypeSymlink:
		return repr_proto.FileInfoProto_TYPE_SYMLINK, nil
	case fs.TypeBlockDevice:
		return repr_proto.FileInfoProto_TYPE_BLOCK_DEVICE, nil
	case fs.TypeCharDevice:
		return repr_proto.FileInfoProto_TYPE_CHAR_DEVICE, nil
	case fs.TypeNamedPipe:
		return repr_proto.FileInfoProto_TYPE_NAMED_PIPE, nil
	}

	return 0, fmt.Errorf("Unrecognized EntryType: %v", t)
}

func makeInfoProto(
	entry *fs.FileInfo) (*repr_proto.FileInfoProto, error) {
	blobs := []*repr_proto.BlobInfoProto{}
	for i, _ := range entry.Scores {
		// Make a copy of the score (a value type, not a reference type), for
		// slicing below.
		var score blob.Score = entry.Scores[i]

		proto := &repr_proto.BlobInfoProto{Hash: score[:]}
		blobs = append(blobs, proto)
	}

	entryProto := &repr_proto.FileInfoProto{
		Permissions:    proto.Uint32(uint32(entry.Permissions)),
		Uid:            proto.Uint32(uint32(entry.Uid)),
		Username:       entry.Username,
		Gid:            proto.Uint32(uint32(entry.Gid)),
		Groupname:      entry.Groupname,
		HardLinkTarget: entry.HardLinkTarget,
		Name:           proto.String(entry.Name),
		Mtime: &repr_proto.TimeProto{
			Second:     proto.Int64(entry.MTime.Unix()),
			Nanosecond: proto.Uint32(uint32(entry.MTime.Nanosecond())),
		},
		Size:  proto.Uint64(entry.Size),
		Inode: proto.Uint64(entry.Inode),
		Blob:  blobs,
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

const (
	magicByte_Dir  byte = 'd'
	magicByte_File byte = 'f'
)

// MarshalDir turns a directory listing into bytes that can later be used with
// UnmarshalDir. Note that ContainingDevice fields are lost.
//
// The input array may be modified.
func MarshalDir(listing []*fs.FileInfo) (d []byte, err error) {
	// Convert to proto buffers.
	protos := []*repr_proto.FileInfoProto{}
	for _, fi := range listing {
		proto, err := makeInfoProto(fi)
		if err != nil {
			return nil, err
		}

		protos = append(protos, proto)
	}

	// Encapsulate the entries into a listing proto and serialize that.
	listingProto := &repr_proto.DirectoryListingProto{Entry: protos}

	d, err = proto.Marshal(listingProto)
	if err != nil {
		err = fmt.Errorf("proto.Marshal: %v", err)
		return
	}

	// Append a magic byte so we can recognize this as a directory without
	// context.
	d = append(d, magicByte_Dir)

	return
}

// MarshalFile encodes the supplied file contents into bytes that can later be
// used with UnmarshalFile. The input array may be modified.
func MarshalFile(contents []byte) (f []byte, err error) {
	// Append a magic byte so we can recognize this as a file without context.
	f = append(contents, magicByte_File)
	return
}

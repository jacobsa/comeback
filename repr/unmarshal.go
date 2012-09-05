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
	"errors"
	"fmt"
	"github.com/jacobsa/comeback/blob"
	"github.com/jacobsa/comeback/fs"
	"github.com/jacobsa/comeback/repr/proto"
	"github.com/jacobsa/comeback/sys"
	"os"
	"time"
)

func convertProtoType(t repr_proto.DirectoryEntryProto_Type) (fs.EntryType, error) {
	switch t {
	case repr_proto.DirectoryEntryProto_TYPE_FILE:
		return fs.TypeFile, nil
	case repr_proto.DirectoryEntryProto_TYPE_DIRECTORY:
		return fs.TypeDirectory, nil
	case repr_proto.DirectoryEntryProto_TYPE_SYMLINK:
		return fs.TypeSymlink, nil
	case repr_proto.DirectoryEntryProto_TYPE_BLOCK_DEVICE:
		return fs.TypeBlockDevice, nil
	case repr_proto.DirectoryEntryProto_TYPE_CHAR_DEVICE:
		return fs.TypeCharDevice, nil
	case repr_proto.DirectoryEntryProto_TYPE_NAMED_PIPE:
		return fs.TypeNamedPipe, nil
	}

	return 0, fmt.Errorf("Unrecognized DirectoryEntryProto_Type: %v", t)
}

func convertTimeProto(timeProto *repr_proto.TimeProto) (time.Time, error) {
	return time.Unix(timeProto.GetSecond(), int64(timeProto.GetNanosecond())), nil
}

func convertBlobInfoProto(p *repr_proto.BlobInfoProto) (blob.Score, error) {
	if len(p.Hash) != 20 {
		return nil, fmt.Errorf("Illegal hash length: %d", len(p.Hash))
	}

	return blob.Score(p.Hash), nil
}

func convertEntryProto(entryProto *repr_proto.DirectoryEntryProto) (entry *fs.DirectoryEntry, err error) {
	entry = &fs.DirectoryEntry{}

	entry.Name = entryProto.GetName()
	entry.Permissions = os.FileMode(entryProto.GetPermissions())
	entry.Uid = sys.UserId(entryProto.GetUid())
	entry.Username = entryProto.Username
	entry.Gid = sys.GroupId(entryProto.GetGid())
	entry.Groupname = entryProto.Groupname
	entry.HardLinkTarget = entryProto.HardLinkTarget

	// Copy symlink targets.
	entry.Target = entryProto.GetTarget()

	// Copy device numbers.
	entry.DeviceNumber = entryProto.GetDeviceNumber()

	// Attempt to convert the type.
	entry.Type, err = convertProtoType(entryProto.GetType())
	if err != nil {
		return nil, err
	}

	// Attempt to convert the modification time.
	entry.MTime, err = convertTimeProto(entryProto.GetMtime())
	if err != nil {
		return nil, err
	}

	// Attempt to convert each score proto.
	for _, blobInfoProto := range entryProto.Blob {
		score, err := convertBlobInfoProto(blobInfoProto)
		if err != nil {
			return nil, err
		}

		entry.Scores = append(entry.Scores, score)
	}

	return entry, nil
}

// Unmarshal recovers a list of directory entries from bytes previously
// returned by Marshal.
func Unmarshal(d []byte) (entries []*fs.DirectoryEntry, err error) {
	// Parse the protocol buffer.
	listingProto := new(repr_proto.DirectoryListingProto)
	err = proto.Unmarshal(d, listingProto)
	if err != nil {
		return nil, errors.New("Parsing data: " + err.Error())
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

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
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jacobsa/comeback/internal/blob"
	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/repr/proto"
	"github.com/jacobsa/comeback/internal/sys"
)

func convertProtoType(
	t repr_proto.FileInfoProto_Type) (fs.Type, error) {
	switch t {
	case repr_proto.FileInfoProto_TYPE_FILE:
		return fs.TypeFile, nil
	case repr_proto.FileInfoProto_TYPE_DIRECTORY:
		return fs.TypeDirectory, nil
	case repr_proto.FileInfoProto_TYPE_SYMLINK:
		return fs.TypeSymlink, nil
	case repr_proto.FileInfoProto_TYPE_BLOCK_DEVICE:
		return fs.TypeBlockDevice, nil
	case repr_proto.FileInfoProto_TYPE_CHAR_DEVICE:
		return fs.TypeCharDevice, nil
	case repr_proto.FileInfoProto_TYPE_NAMED_PIPE:
		return fs.TypeNamedPipe, nil
	}

	return 0, fmt.Errorf("Unrecognized FileInfoProto_Type: %v", t)
}

func convertTimeProto(timeProto *repr_proto.TimeProto) (time.Time, error) {
	return time.Unix(timeProto.GetSecond(), int64(timeProto.GetNanosecond())), nil
}

func convertBlobInfoProto(
	p *repr_proto.BlobInfoProto) (s blob.Score, err error) {
	if len(p.Hash) != blob.ScoreLength {
		err = fmt.Errorf("Illegal hash length: %d", len(p.Hash))
		return
	}

	copy(s[:], p.Hash)
	return
}

func convertEntryProto(
	entryProto *repr_proto.FileInfoProto) (
	entry *fs.FileInfo,
	err error) {
	entry = &fs.FileInfo{}

	entry.Name = entryProto.GetName()
	entry.Permissions = os.FileMode(entryProto.GetPermissions())
	entry.Uid = sys.UserId(entryProto.GetUid())
	entry.Username = entryProto.Username
	entry.Gid = sys.GroupId(entryProto.GetGid())
	entry.Groupname = entryProto.Groupname
	entry.Size = entryProto.GetSize()
	entry.Inode = entryProto.GetInode()
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

// UnmarshalDir recovers a list of directory entries from bytes previously
// returned by MarshalDir.
func UnmarshalDir(d []byte) (entries []*fs.FileInfo, err error) {
	// Verify and strip the magic byte.
	l := len(d)
	if l == 0 || d[l-1] != magicByte_Dir {
		err = fmt.Errorf("Not a directory")
		return
	}

	d = d[:l-1]

	// Parse the protocol buffer.
	listingProto := new(repr_proto.DirectoryListingProto)
	err = proto.Unmarshal(d, listingProto)
	if err != nil {
		return nil, errors.New("Parsing data: " + err.Error())
	}

	// Convert each entry.
	entries = []*fs.FileInfo{}
	for _, entryProto := range listingProto.Entry {
		entry, err := convertEntryProto(entryProto)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return
}

// UnmarshalFile recovers file contents previously encoded with MarshalFile.
func UnmarshalFile(f []byte) (contents []byte, err error) {
	// Verify and strip the magic byte.
	l := len(f)
	if l == 0 || f[l-1] != magicByte_File {
		err = fmt.Errorf("Not a file")
		return
	}

	contents = f[:l-1]
	return
}

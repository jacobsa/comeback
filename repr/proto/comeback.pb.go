// Code generated by protoc-gen-go.
// source: comeback.proto
// DO NOT EDIT!

/*
Package repr_proto is a generated protocol buffer package.

It is generated from these files:
	comeback.proto

It has these top-level messages:
	BlobInfoProto
	TimeProto
	DirectoryEntryProto
	DirectoryListingProto
*/
package repr_proto

import proto "github.com/golang/protobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type DirectoryEntryProto_Type int32

const (
	// Sentinel for a missing value.
	DirectoryEntryProto_TYPE_UNKNOWN      DirectoryEntryProto_Type = -1
	DirectoryEntryProto_TYPE_FILE         DirectoryEntryProto_Type = 0
	DirectoryEntryProto_TYPE_DIRECTORY    DirectoryEntryProto_Type = 1
	DirectoryEntryProto_TYPE_SYMLINK      DirectoryEntryProto_Type = 2
	DirectoryEntryProto_TYPE_BLOCK_DEVICE DirectoryEntryProto_Type = 3
	DirectoryEntryProto_TYPE_CHAR_DEVICE  DirectoryEntryProto_Type = 4
	DirectoryEntryProto_TYPE_NAMED_PIPE   DirectoryEntryProto_Type = 5
)

var DirectoryEntryProto_Type_name = map[int32]string{
	-1: "TYPE_UNKNOWN",
	0:  "TYPE_FILE",
	1:  "TYPE_DIRECTORY",
	2:  "TYPE_SYMLINK",
	3:  "TYPE_BLOCK_DEVICE",
	4:  "TYPE_CHAR_DEVICE",
	5:  "TYPE_NAMED_PIPE",
}
var DirectoryEntryProto_Type_value = map[string]int32{
	"TYPE_UNKNOWN":      -1,
	"TYPE_FILE":         0,
	"TYPE_DIRECTORY":    1,
	"TYPE_SYMLINK":      2,
	"TYPE_BLOCK_DEVICE": 3,
	"TYPE_CHAR_DEVICE":  4,
	"TYPE_NAMED_PIPE":   5,
}

func (x DirectoryEntryProto_Type) Enum() *DirectoryEntryProto_Type {
	p := new(DirectoryEntryProto_Type)
	*p = x
	return p
}
func (x DirectoryEntryProto_Type) String() string {
	return proto.EnumName(DirectoryEntryProto_Type_name, int32(x))
}
func (x *DirectoryEntryProto_Type) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(DirectoryEntryProto_Type_value, data, "DirectoryEntryProto_Type")
	if err != nil {
		return err
	}
	*x = DirectoryEntryProto_Type(value)
	return nil
}

type BlobInfoProto struct {
	// The SHA-1 hash of the blob, in 20-byte 'raw' form.
	Hash             []byte `protobuf:"bytes,1,opt,name=hash" json:"hash,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *BlobInfoProto) Reset()         { *m = BlobInfoProto{} }
func (m *BlobInfoProto) String() string { return proto.CompactTextString(m) }
func (*BlobInfoProto) ProtoMessage()    {}

func (m *BlobInfoProto) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

// An instant in time, with nanosecond resolution.
type TimeProto struct {
	// The number of seconds since the Unix time epoch.
	Second *int64 `protobuf:"varint,1,opt,name=second" json:"second,omitempty"`
	// A nanosecond in the second specified above. This must be in the range
	// [0, 1e9).
	Nanosecond       *uint32 `protobuf:"varint,2,opt,name=nanosecond" json:"nanosecond,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *TimeProto) Reset()         { *m = TimeProto{} }
func (m *TimeProto) String() string { return proto.CompactTextString(m) }
func (*TimeProto) ProtoMessage()    {}

func (m *TimeProto) GetSecond() int64 {
	if m != nil && m.Second != nil {
		return *m.Second
	}
	return 0
}

func (m *TimeProto) GetNanosecond() uint32 {
	if m != nil && m.Nanosecond != nil {
		return *m.Nanosecond
	}
	return 0
}

type DirectoryEntryProto struct {
	Type *DirectoryEntryProto_Type `protobuf:"varint,1,opt,name=type,enum=repr_proto.DirectoryEntryProto_Type" json:"type,omitempty"`
	// The permissions bits for this entry, as described in the documentation for
	// fs.DirectoryEntry, with values as in the golang os package.
	Permissions *uint32 `protobuf:"varint,2,opt,name=permissions" json:"permissions,omitempty"`
	// The owning user's UID, and their username if known.
	Uid      *uint32 `protobuf:"varint,3,opt,name=uid" json:"uid,omitempty"`
	Username *string `protobuf:"bytes,4,opt,name=username" json:"username,omitempty"`
	// The owning user's GID, and their groupname if known.
	Gid       *uint32 `protobuf:"varint,5,opt,name=gid" json:"gid,omitempty"`
	Groupname *string `protobuf:"bytes,6,opt,name=groupname" json:"groupname,omitempty"`
	// The name of this entry.
	Name *string `protobuf:"bytes,7,opt,name=name" json:"name,omitempty"`
	// The modification time.
	Mtime *TimeProto `protobuf:"bytes,8,opt,name=mtime" json:"mtime,omitempty"`
	// The zero or more blobs that make up a regular file's contents, to be
	// concatenated in order. Scores are present only if hard_link_target is not
	// present.
	Blob []*BlobInfoProto `protobuf:"bytes,9,rep,name=blob" json:"blob,omitempty"`
	// If this entry belongs to a backup containing another file to which it is
	// hard linked, this is the target of the hard link relative to the root of
	// the backup.
	HardLinkTarget *string `protobuf:"bytes,10,opt,name=hard_link_target" json:"hard_link_target,omitempty"`
	// The target, if this is a symlink.
	Target *string `protobuf:"bytes,11,opt,name=target" json:"target,omitempty"`
	// The device number in a system-dependent format, if this is a device.
	DeviceNumber     *int32 `protobuf:"varint,12,opt,name=device_number" json:"device_number,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *DirectoryEntryProto) Reset()         { *m = DirectoryEntryProto{} }
func (m *DirectoryEntryProto) String() string { return proto.CompactTextString(m) }
func (*DirectoryEntryProto) ProtoMessage()    {}

func (m *DirectoryEntryProto) GetType() DirectoryEntryProto_Type {
	if m != nil && m.Type != nil {
		return *m.Type
	}
	return DirectoryEntryProto_TYPE_UNKNOWN
}

func (m *DirectoryEntryProto) GetPermissions() uint32 {
	if m != nil && m.Permissions != nil {
		return *m.Permissions
	}
	return 0
}

func (m *DirectoryEntryProto) GetUid() uint32 {
	if m != nil && m.Uid != nil {
		return *m.Uid
	}
	return 0
}

func (m *DirectoryEntryProto) GetUsername() string {
	if m != nil && m.Username != nil {
		return *m.Username
	}
	return ""
}

func (m *DirectoryEntryProto) GetGid() uint32 {
	if m != nil && m.Gid != nil {
		return *m.Gid
	}
	return 0
}

func (m *DirectoryEntryProto) GetGroupname() string {
	if m != nil && m.Groupname != nil {
		return *m.Groupname
	}
	return ""
}

func (m *DirectoryEntryProto) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *DirectoryEntryProto) GetMtime() *TimeProto {
	if m != nil {
		return m.Mtime
	}
	return nil
}

func (m *DirectoryEntryProto) GetBlob() []*BlobInfoProto {
	if m != nil {
		return m.Blob
	}
	return nil
}

func (m *DirectoryEntryProto) GetHardLinkTarget() string {
	if m != nil && m.HardLinkTarget != nil {
		return *m.HardLinkTarget
	}
	return ""
}

func (m *DirectoryEntryProto) GetTarget() string {
	if m != nil && m.Target != nil {
		return *m.Target
	}
	return ""
}

func (m *DirectoryEntryProto) GetDeviceNumber() int32 {
	if m != nil && m.DeviceNumber != nil {
		return *m.DeviceNumber
	}
	return 0
}

type DirectoryListingProto struct {
	Entry            []*DirectoryEntryProto `protobuf:"bytes,1,rep,name=entry" json:"entry,omitempty"`
	XXX_unrecognized []byte                 `json:"-"`
}

func (m *DirectoryListingProto) Reset()         { *m = DirectoryListingProto{} }
func (m *DirectoryListingProto) String() string { return proto.CompactTextString(m) }
func (*DirectoryListingProto) ProtoMessage()    {}

func (m *DirectoryListingProto) GetEntry() []*DirectoryEntryProto {
	if m != nil {
		return m.Entry
	}
	return nil
}

func init() {
	proto.RegisterEnum("repr_proto.DirectoryEntryProto_Type", DirectoryEntryProto_Type_name, DirectoryEntryProto_Type_value)
}

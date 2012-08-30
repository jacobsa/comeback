// Code generated by protoc-gen-go from "repr/proto/comeback.proto"
// DO NOT EDIT!

package repr_proto

import proto "code.google.com/p/goprotobuf/proto"
import json "encoding/json"
import math "math"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type DirectoryEntryProto_Type int32

const (
	DirectoryEntryProto_TYPE_UNKNOWN   DirectoryEntryProto_Type = -1
	DirectoryEntryProto_TYPE_FILE      DirectoryEntryProto_Type = 0
	DirectoryEntryProto_TYPE_DIRECTORY DirectoryEntryProto_Type = 1
	DirectoryEntryProto_TYPE_SYMLINK   DirectoryEntryProto_Type = 2
)

var DirectoryEntryProto_Type_name = map[int32]string{
	-1: "TYPE_UNKNOWN",
	0:  "TYPE_FILE",
	1:  "TYPE_DIRECTORY",
	2:  "TYPE_SYMLINK",
}
var DirectoryEntryProto_Type_value = map[string]int32{
	"TYPE_UNKNOWN":   -1,
	"TYPE_FILE":      0,
	"TYPE_DIRECTORY": 1,
	"TYPE_SYMLINK":   2,
}

func (x DirectoryEntryProto_Type) Enum() *DirectoryEntryProto_Type {
	p := new(DirectoryEntryProto_Type)
	*p = x
	return p
}
func (x DirectoryEntryProto_Type) String() string {
	return proto.EnumName(DirectoryEntryProto_Type_name, int32(x))
}
func (x DirectoryEntryProto_Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
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
	Hash             []byte `protobuf:"bytes,1,opt,name=hash" json:"hash,omitempty"`
	XXX_unrecognized []byte `json:"-"`
}

func (this *BlobInfoProto) Reset()         { *this = BlobInfoProto{} }
func (this *BlobInfoProto) String() string { return proto.CompactTextString(this) }
func (*BlobInfoProto) ProtoMessage()       {}

func (this *BlobInfoProto) GetHash() []byte {
	if this != nil {
		return this.Hash
	}
	return nil
}

type TimeProto struct {
	Second           *int64  `protobuf:"varint,1,opt,name=second" json:"second,omitempty"`
	Nanosecond       *uint32 `protobuf:"varint,2,opt,name=nanosecond" json:"nanosecond,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (this *TimeProto) Reset()         { *this = TimeProto{} }
func (this *TimeProto) String() string { return proto.CompactTextString(this) }
func (*TimeProto) ProtoMessage()       {}

func (this *TimeProto) GetSecond() int64 {
	if this != nil && this.Second != nil {
		return *this.Second
	}
	return 0
}

func (this *TimeProto) GetNanosecond() uint32 {
	if this != nil && this.Nanosecond != nil {
		return *this.Nanosecond
	}
	return 0
}

type DirectoryEntryProto struct {
	Type             *DirectoryEntryProto_Type `protobuf:"varint,1,opt,name=type,enum=repr_proto.DirectoryEntryProto_Type" json:"type,omitempty"`
	Permissions      *uint32                   `protobuf:"varint,2,opt,name=permissions" json:"permissions,omitempty"`
	Name             *string                   `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
	Mtime            *TimeProto                `protobuf:"bytes,4,opt,name=mtime" json:"mtime,omitempty"`
	Blob             []*BlobInfoProto          `protobuf:"bytes,5,rep,name=blob" json:"blob,omitempty"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (this *DirectoryEntryProto) Reset()         { *this = DirectoryEntryProto{} }
func (this *DirectoryEntryProto) String() string { return proto.CompactTextString(this) }
func (*DirectoryEntryProto) ProtoMessage()       {}

func (this *DirectoryEntryProto) GetType() DirectoryEntryProto_Type {
	if this != nil && this.Type != nil {
		return *this.Type
	}
	return 0
}

func (this *DirectoryEntryProto) GetPermissions() uint32 {
	if this != nil && this.Permissions != nil {
		return *this.Permissions
	}
	return 0
}

func (this *DirectoryEntryProto) GetName() string {
	if this != nil && this.Name != nil {
		return *this.Name
	}
	return ""
}

func (this *DirectoryEntryProto) GetMtime() *TimeProto {
	if this != nil {
		return this.Mtime
	}
	return nil
}

type DirectoryListingProto struct {
	Entry            []*DirectoryEntryProto `protobuf:"bytes,1,rep,name=entry" json:"entry,omitempty"`
	XXX_unrecognized []byte                 `json:"-"`
}

func (this *DirectoryListingProto) Reset()         { *this = DirectoryListingProto{} }
func (this *DirectoryListingProto) String() string { return proto.CompactTextString(this) }
func (*DirectoryListingProto) ProtoMessage()       {}

func init() {
	proto.RegisterEnum("repr_proto.DirectoryEntryProto_Type", DirectoryEntryProto_Type_name, DirectoryEntryProto_Type_value)
}
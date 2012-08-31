// Copied and modified from the golang os/user package on 2012-08-31, Go
// version 1.0.2. Modifications copyright 2012 Aaron Jacobs; see the comeback
// LICENSE file. Original copyright notice below.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin freebsd linux
// +build cgo

package group

import (
	"fmt"
	"runtime"
	"strconv"
	"syscall"
	"unsafe"
)

/*
#include <unistd.h>
#include <sys/types.h>
#include <grp.h>
#include <stdlib.h>
*/
import "C"

// Current returns the current group. 
func Current() (*Group, error) {
	return lookup(syscall.Getgid(), "", false)
}

// Lookup looks up a group by groupname. If the group cannot be found,
// the returned error is of type UnknownGroupError.
func Lookup(groupname string) (*Group, error) {
	return lookup(-1, groupname, true)
}

// LookupId looks up a group by groupid. If the group cannot be found,
// the returned error is of type UnknownGroupIdError.
func LookupId(gid string) (*Group, error) {
	i, e := strconv.Atoi(gid)
	if e != nil {
		return nil, e
	}
	return lookup(i, "", false)
}

func lookup(gid int, groupname string, lookupByName bool) (*Group, error) {
	var grp C.struct_group
	var result *C.struct_group

	var bufSize C.long
	if runtime.GOOS == "freebsd" {
		panic("Don't know how to deal with freebsd.")
	} else {
		bufSize = C.sysconf(C._SC_GETGR_R_SIZE_MAX)
		if bufSize <= 0 || bufSize > 1<<20 {
			return nil, fmt.Errorf("group: unreasonable _SC_GETGR_R_SIZE_MAX of %d", bufSize)
		}
	}
	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)
	var rv C.int
	if lookupByName {
		nameC := C.CString(groupname)
		defer C.free(unsafe.Pointer(nameC))
		rv = C.getgrnam_r(nameC,
			&grp,
			(*C.char)(buf),
			C.size_t(bufSize),
			&result)
		if rv != 0 {
			return nil, fmt.Errorf("group: lookup groupname %s: %s", groupname, syscall.Errno(rv))
		}
		if result == nil {
			return nil, UnknownGroupError(groupname)
		}
	} else {
		rv = C.getgrgid_r(C.gid_t(gid),
			&grp,
			(*C.char)(buf),
			C.size_t(bufSize),
			&result)
		if rv != 0 {
			return nil, fmt.Errorf("group: lookup groupid %d: %s", gid, syscall.Errno(rv))
		}
		if result == nil {
			return nil, UnknownGroupIdError(gid)
		}
	}
	u := &Group{
		Gid:       strconv.Itoa(int(grp.gr_gid)),
		Groupname: C.GoString(grp.gr_name),
	}
	return u, nil
}

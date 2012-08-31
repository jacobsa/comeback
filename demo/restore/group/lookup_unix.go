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
#include <pwd.h>
#include <stdlib.h>

static int mygetpwuid_r(int uid, struct passwd *pwd,
	char *buf, size_t buflen, struct passwd **result) {
 return getpwuid_r(uid, pwd, buf, buflen, result);
}
*/
import "C"

// Current returns the current group. 
func Current() (*Group, error) {
	return lookup(syscall.Getuid(), "", false)
}

// Lookup looks up a group by groupname. If the group cannot be found,
// the returned error is of type UnknownGroupError.
func Lookup(groupname string) (*Group, error) {
	return lookup(-1, groupname, true)
}

// LookupId looks up a group by groupid. If the group cannot be found,
// the returned error is of type UnknownGroupIdError.
func LookupId(uid string) (*Group, error) {
	i, e := strconv.Atoi(uid)
	if e != nil {
		return nil, e
	}
	return lookup(i, "", false)
}

func lookup(uid int, groupname string, lookupByName bool) (*Group, error) {
	var pwd C.struct_passwd
	var result *C.struct_passwd

	var bufSize C.long
	if runtime.GOOS == "freebsd" {
		// FreeBSD doesn't have _SC_GETPW_R_SIZE_MAX
		// and just returns -1.  So just use the same
		// size that Linux returns
		bufSize = 1024
	} else {
		bufSize = C.sysconf(C._SC_GETPW_R_SIZE_MAX)
		if bufSize <= 0 || bufSize > 1<<20 {
			return nil, fmt.Errorf("group: unreasonable _SC_GETPW_R_SIZE_MAX of %d", bufSize)
		}
	}
	buf := C.malloc(C.size_t(bufSize))
	defer C.free(buf)
	var rv C.int
	if lookupByName {
		nameC := C.CString(groupname)
		defer C.free(unsafe.Pointer(nameC))
		rv = C.getpwnam_r(nameC,
			&pwd,
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
		// mygetpwuid_r is a wrapper around getpwuid_r to
		// to avoid using uid_t because C.uid_t(uid) for
		// unknown reasons doesn't work on linux.
		rv = C.mygetpwuid_r(C.int(uid),
			&pwd,
			(*C.char)(buf),
			C.size_t(bufSize),
			&result)
		if rv != 0 {
			return nil, fmt.Errorf("group: lookup groupid %d: %s", uid, syscall.Errno(rv))
		}
		if result == nil {
			return nil, UnknownGroupIdError(uid)
		}
	}
	u := &Group{
		Gid:      strconv.Itoa(int(pwd.pw_gid)),
		Groupname: C.GoString(pwd.pw_name),
	}
	return u, nil
}

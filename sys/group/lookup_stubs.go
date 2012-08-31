// Copied and modified from the golang os/user package on 2012-08-31, Go
// version 1.0.2. Modifications copyright 2012 Aaron Jacobs; see the comeback
// LICENSE file. Original copyright notice below.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !cgo,!windows

package group

import (
	"fmt"
	"runtime"
)

func init() {
	implemented = false
}

func Current() (*group, error) {
	return nil, fmt.Errorf("group: Current not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func Lookup(groupname string) (*Group, error) {
	return nil, fmt.Errorf("group: Lookup not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

func LookupId(string) (*group, error) {
	return nil, fmt.Errorf("group: LookupId not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

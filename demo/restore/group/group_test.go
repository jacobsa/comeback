// Copied and modified from the golang os/user package on 2012-08-31, Go
// version 1.0.2. Modifications copyright 2012 Aaron Jacobs; see the comeback
// LICENSE file. Original copyright notice below.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package group

import (
	"runtime"
	"testing"
)

func skip(t *testing.T) bool {
	if !implemented {
		t.Logf("group: not implemented; skipping tests")
		return true
	}

	switch runtime.GOOS {
	case "linux", "freebsd", "darwin", "windows":
		return false
	}

	t.Logf("group: Lookup not implemented on %s; skipping test", runtime.GOOS)
	return true
}

func TestCurrent(t *testing.T) {
	if skip(t) {
		return
	}

	g, err := Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if g.Groupname == "" {
		t.Fatalf("didn't get a groupname")
	}
}

func compare(t *testing.T, want, got *Group) {
	if want.Groupname != got.Groupname {
		t.Errorf("got Groupname=%q; want %q", got.Groupname, want.Groupname)
	}
	if want.Gid != got.Gid {
		t.Errorf("got Gid=%q; want %q", got.Gid, want.Gid)
	}
}

func TestLookup(t *testing.T) {
	if skip(t) {
		return
	}

	want, err := Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	got, err := Lookup(want.Groupname)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	compare(t, want, got)
}

func TestLookupId(t *testing.T) {
	if skip(t) {
		return
	}

	want, err := Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	got, err := LookupId(want.Gid)
	if err != nil {
		t.Fatalf("LookupId: %v", err)
	}
	compare(t, want, got)
}

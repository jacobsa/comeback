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

package fs_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/jacobsa/comeback/internal/fs"
	"github.com/jacobsa/comeback/internal/sys"
	"github.com/jacobsa/comeback/internal/sys/group"
	. "github.com/jacobsa/oglematchers"
	. "github.com/jacobsa/ogletest"
)

func TestConvertFileInfo(t *testing.T) { RunTests(t) }

////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////

// Like os.Chmod, but don't follow symlinks.
func setPermissions(path string, permissions uint32) error {
	// Open the file without following symlinks.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call fchmod.
	err = syscall.Fchmod(fd, permissions)
	if err != nil {
		return err
	}

	return nil
}

// Create a named pipe at the supplied path.
func makeNamedPipe(path string, permissions uint32) error {
	return syscall.Mkfifo(path, permissions)
}

// Set the modification time for the supplied path without following symlinks
// (as syscall.Chtimes and therefore os.Chtimes do).
//
// Cf. http://stackoverflow.com/a/10611073
func setModTime(path string, mtime time.Time) error {
	// Open the file without following symlinks. Use O_NONBLOCK to allow opening
	// of named pipes without a writer.
	fd, err := syscall.Open(path, syscall.O_NONBLOCK|syscall.O_SYMLINK, 0)
	if err != nil {
		return err
	}

	defer syscall.Close(fd)

	// Call futimes.
	var utimes [2]syscall.Timeval
	atime := time.Now()
	atime_ns := atime.UnixNano()
	mtime_ns := mtime.UnixNano()
	utimes[0] = syscall.NsecToTimeval(atime_ns)
	utimes[1] = syscall.NsecToTimeval(mtime_ns)

	err = syscall.Futimes(fd, utimes[0:])
	if err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////
// Boilerplate
////////////////////////////////////////////////////////////////////////

type ConvertFileInfoTest struct {
	// Information about the running process.
	myUid       sys.UserId
	myUsername  string
	myGid       sys.GroupId
	myGroupname string

	// A temporary directory that will be deleted when the test completes.
	baseDir                 string
	baseDirContainingDevice int32

	// The path to be stat'd, and the resulting struct.
	path string
	info *fs.FileInfo
}

var _ SetUpInterface = &ConvertFileInfoTest{}
var _ TearDownInterface = &ConvertFileInfoTest{}

func init() { RegisterTestSuite(&ConvertFileInfoTest{}) }

func (t *ConvertFileInfoTest) SetUp(ti *TestInfo) {
	var err error

	// Find user info.
	currentUser, err := user.Current()
	AssertEq(nil, err)

	currentUserId, err := strconv.Atoi(currentUser.Uid)
	AssertEq(nil, err)

	t.myUid = sys.UserId(currentUserId)
	t.myUsername = currentUser.Username
	AssertNe(0, t.myUid)
	AssertNe("", t.myUsername)

	// Find group info.
	currentGroup, err := group.Current()
	AssertEq(nil, err)

	currentGroupId, err := strconv.Atoi(currentGroup.Gid)
	AssertEq(nil, err)

	t.myGid = sys.GroupId(currentGroupId)
	t.myGroupname = currentGroup.Groupname
	AssertNe(0, t.myGid)
	AssertNe("", t.myGroupname)

	// Set up the temporary directory.
	t.baseDir, err = ioutil.TempDir("", "convert_file_info_test")
	AssertEq(nil, err)

	// Find its containing device.
	fi, err := os.Stat(t.baseDir)
	AssertEq(nil, err)

	t.baseDirContainingDevice = fi.Sys().(*syscall.Stat_t).Dev
}

func (t *ConvertFileInfoTest) TearDown() {
	err := os.RemoveAll(t.baseDir)
	AssertEq(nil, err)
}

func (t *ConvertFileInfoTest) call() (err error) {
	// Stat the path.
	fi, err := os.Lstat(t.path)
	if err != nil {
		err = fmt.Errorf("Stat: %v", err)
		return
	}

	// Read the symlink target if necessary.
	var symlinkTarget string
	if fi.Mode()&os.ModeSymlink != 0 {
		symlinkTarget, err = os.Readlink(t.path)
		if err != nil {
			err = fmt.Errorf("Readlink: %v", err)
			return
		}
	}

	// Call through.
	t.info, err = fs.ConvertFileInfo(fi, symlinkTarget)
	if err != nil {
		err = fmt.Errorf("ConvertFileInfo: %v", err)
		return
	}

	return
}

////////////////////////////////////////////////////////////////////////
// Tests
////////////////////////////////////////////////////////////////////////

func (t *ConvertFileInfoTest) RegularFile() {
	var err error

	// File
	t.path = path.Join(t.baseDir, "burrito.txt")
	err = ioutil.WriteFile(t.path, []byte("queso"), 0600)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeFile, t.info.Type)
	ExpectEq("burrito.txt", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(0, t.info.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, t.info.Permissions)
	ExpectEq(t.myUid, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.info.MTime.Equal(mtime), "%v", t.info.MTime)
	ExpectEq(len("queso"), t.info.Size)
	ExpectThat(t.info.Scores, ElementsAre())

	AssertNe(0, t.info.Inode)
	ExpectEq(t.baseDirContainingDevice, t.info.ContainingDevice)
}

func (t *ConvertFileInfoTest) Directory() {
	var err error

	// Dir
	t.path = path.Join(t.baseDir, "burrito")
	err = os.Mkdir(t.path, 0700)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0751|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	fi, err := os.Stat(t.path)
	AssertEq(nil, err)
	stat := fi.Sys().(*syscall.Stat_t)
	AssertNe(0, stat.Ino)

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeDirectory, t.info.Type)
	ExpectEq("burrito", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(0, t.info.DeviceNumber)
	ExpectEq(0751|os.ModeSetgid, t.info.Permissions)
	ExpectEq(t.myUid, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.info.MTime.Equal(mtime), "%v", t.info.MTime)
	ExpectEq(stat.Size, t.info.Size)
	ExpectEq(stat.Dev, t.info.ContainingDevice)
	ExpectEq(stat.Ino, t.info.Inode)
	ExpectThat(t.info.Scores, ElementsAre())
}

func (t *ConvertFileInfoTest) Symlinks() {
	var err error

	// Link
	t.path = path.Join(t.baseDir, "burrito")
	err = os.Symlink("/foo/burrito", t.path)
	AssertEq(nil, err)

	err = setPermissions(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeSymlink, t.info.Type)
	ExpectEq("burrito", t.info.Name)
	ExpectEq("/foo/burrito", t.info.Target)
	ExpectEq(0, t.info.DeviceNumber)
	ExpectEq(0714|os.ModeSetgid, t.info.Permissions)
	ExpectEq(t.myUid, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.info.MTime.Equal(mtime), "%v", t.info.MTime)
	ExpectThat(t.info.Scores, ElementsAre())
}

func (t *ConvertFileInfoTest) CharDevices() {
	var err error
	t.path = "/dev/urandom"

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeCharDevice, t.info.Type)
	ExpectEq("urandom", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(os.FileMode(0666), t.info.Permissions)
	ExpectEq(0, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals("root")))
	ExpectEq(0, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals("wheel")))
	ExpectGe(time.Since(t.info.MTime), 0)
	ExpectLt(time.Since(t.info.MTime), 365*24*time.Hour)
}

func (t *ConvertFileInfoTest) BlockDevices() {
	var err error
	t.path = "/dev/disk0"

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeBlockDevice, t.info.Type)
	ExpectEq("disk0", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(os.FileMode(0640), t.info.Permissions)
	ExpectEq(0, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals("root")))
	ExpectEq(5, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals("operator")))
	ExpectGe(time.Since(t.info.MTime), 0)
	ExpectLt(time.Since(t.info.MTime), 365*24*time.Hour)
}

func (t *ConvertFileInfoTest) NamedPipes() {
	var err error

	// Pipe
	t.path = path.Join(t.baseDir, "burrito")
	err = makeNamedPipe(t.path, 0714|syscall.S_ISGID)
	AssertEq(nil, err)

	mtime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	err = setModTime(t.path, mtime)
	AssertEq(nil, err)

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeNamedPipe, t.info.Type)
	ExpectEq("burrito", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(0714|os.ModeSetgid, t.info.Permissions)
	ExpectEq(t.myUid, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectTrue(t.info.MTime.Equal(mtime), "%v", t.info.MTime)
	ExpectThat(t.info.Scores, ElementsAre())
}

func (t *ConvertFileInfoTest) Sockets() {
	var err error

	// Create
	t.path = path.Join(t.baseDir, "burrito")
	listener, err := net.Listen("unix", t.path)
	AssertEq(nil, err)
	defer listener.Close()

	// Call
	err = t.call()

	AssertEq(nil, err)

	ExpectEq(fs.TypeSocket, t.info.Type)
	ExpectEq("burrito", t.info.Name)
	ExpectEq("", t.info.Target)
	ExpectEq(t.myUid, t.info.Uid)
	ExpectThat(t.info.Username, Pointee(Equals(t.myUsername)))
	ExpectEq(t.myGid, t.info.Gid)
	ExpectThat(t.info.Groupname, Pointee(Equals(t.myGroupname)))
	ExpectThat(t.info.Scores, ElementsAre())
}

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

// Package sys contains types and functions useful for finding system
// information.
package sys

import (
	"fmt"
	"strconv"

	"github.com/jacobsa/comeback/internal/sys/group"
)

// A system group ID, aka GID.
type GroupId uint32

// GroupRegistry represents an object that knows about user account group names
// and IDs.
type GroupRegistry interface {
	FindById(id GroupId) (string, error)
	FindByName(name string) (GroupId, error)
}

// Return a group registry hooked up to the system's real group registry.
func NewGroupRegistry() (GroupRegistry, error) {
	return &groupRegistry{}, nil
}

type groupRegistry struct{}

func (r *groupRegistry) FindById(id GroupId) (string, error) {
	osResult, err := group.LookupId(strconv.FormatUint(uint64(id), 10))

	if unknownErr, ok := err.(group.UnknownGroupIdError); ok {
		return "", NotFoundError(unknownErr.Error())
	}

	if err != nil {
		return "", err
	}

	return osResult.Groupname, nil
}

func (r *groupRegistry) FindByName(name string) (GroupId, error) {
	osResult, err := group.Lookup(name)

	if unknownErr, ok := err.(group.UnknownGroupError); ok {
		return 0, NotFoundError(unknownErr.Error())
	}

	if err != nil {
		return 0, err
	}

	// Attempt to parse the GID.
	gid, err := strconv.Atoi(osResult.Gid)
	if err != nil {
		return 0, fmt.Errorf("Unexpected GID format: %s", osResult.Gid)
	}

	return GroupId(gid), nil
}

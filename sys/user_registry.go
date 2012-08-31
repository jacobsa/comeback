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
package fs

import (
)

// A system user ID, aka UID.
type UserId uint32

// UserRegistry represents an object that knows about user account names and
// IDs.
type UserRegistry interface {
	FindById(id UserId) (string, err)
	FindByName(name string) (UserId, err)
}

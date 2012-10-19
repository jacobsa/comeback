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

package main

import (
	"math/rand"
	"sync"
)

var g_randSrcOnce sync.Once
var g_randSrc *rand.Rand

func initRandSrc()

func getRandSrc() *rand.Rand {
	g_randSrcOnce.Do(initRandSrc)
	return g_randSrc
}

func randUint64(randSrc *rand.Rand) uint64 {
	return (uint64(randSrc.Uint32()) << 32) | uint64(randSrc.Uint32())
}

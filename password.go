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

/*
#include <termios.h>
*/
import "C"

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func getTermSettings() (settings C.struct_termios) {
	res := C.tcgetattr(C.int(os.Stdout.Fd()), &settings)
	if res != 0 {
		panic(res)
	}

	return
}

func setTermSettings(settings C.struct_termios) {
	res := C.tcsetattr(C.int(os.Stdout.Fd()), 0, &settings)
	if res != 0 {
		panic(res)
	}
}

func readPassword(prompt string) string {
	// Grab the current terminal settings, making sure they are later restored.
	origTermSettings := getTermSettings()
	defer setTermSettings(origTermSettings)

	// Disable echoing.
	newTermSettings := origTermSettings
	newTermSettings.c_lflag = newTermSettings.c_lflag & ^C.tcflag_t(C.ECHO)
	setTermSettings(newTermSettings)

	// Display the prompt.
	fmt.Print(prompt)

	// Read from stdin. Add a newline for the user pressing enter.
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	fmt.Println("")
	if err != nil {
		log.Fatalln("ReadString:", err)
	}

	return line[0:len(line)-1]
}

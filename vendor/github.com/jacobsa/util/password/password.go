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

package password

/*
#include <termios.h>
*/
import "C"

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
)

// Thin wrapper around tcgetattr.
func getTermSettings() (settings C.struct_termios) {
	res, err := C.tcgetattr(C.int(os.Stderr.Fd()), &settings)
	if res != 0 {
		panic(fmt.Sprintf("tcgetattr returned %d: %v", res, err))
	}

	return
}

// Thin wrapper around tcsetattr.
func setTermSettings(settings C.struct_termios) {
	res, err := C.tcsetattr(C.int(os.Stderr.Fd()), 0, &settings)
	if res != 0 {
		panic(fmt.Sprintf("tcsetattr returned %d: %v", res, err))
	}
}

// A handler for SIGINT that calls the supplied function before terminating.
func handleInterrupt(c <-chan os.Signal, f func()) {
	// Wait for a signal.
	<-c
	f()

	// c.f. http://stackoverflow.com/questions/1101957/are-there-any-standard-exit-status-codes-in-linux
	os.Exit(-1)
}

// Read a password from the terminal without echoing it. No space is added
// after the prompt.
//
// Don't mistake this function for portable; it probably doesn't work anywhere
// except OS X.
func ReadPassword(prompt string) string {
	// Grab the current terminal settings.
	origTermSettings := getTermSettings()

	// Set up a function that will restore the terminal settings exactly once.
	var restoreOnce sync.Once
	restore := func() {
		restoreOnce.Do(func() { setTermSettings(origTermSettings) })
	}

	// Make sure that the settings are restored if we return normally.
	defer restore()

	// Also make sure the settings are restored if the user hits Ctrl-C while the
	// password is being read. The signal handler remains running even after
	// we're done because there is no way to re-enable the default signal
	// handler.
	signalChan := make(chan os.Signal)
	go handleInterrupt(signalChan, restore)
	signal.Notify(signalChan, os.Interrupt)

	// Disable echoing.
	newTermSettings := origTermSettings
	newTermSettings.c_lflag = newTermSettings.c_lflag & ^C.tcflag_t(C.ECHO)
	setTermSettings(newTermSettings)

	// Display the prompt.
	fmt.Fprint(os.Stderr, prompt)

	// Read from stdin. Add a newline for the user pressing enter.
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	fmt.Fprintln(os.Stderr, "")
	if err != nil {
		log.Fatalln("ReadString:", err)
	}

	return line[0 : len(line)-1]
}

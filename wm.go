// Copyright 2017 Sergio Correia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	socket net.Conn
)

func parseReceivedData(recv []byte) {
	received := string(bytes.Trim(recv, "\x00"))
	tokens := strings.Split(received, " ")

	switch strings.ToUpper(tokens[0]) {
	case "TOGGLE-BAR":
		if monitor, err := strconv.Atoi(tokens[1]); err == nil {
			toggleBars(monitor)
		} else {
			fmt.Println(err)
		}
	}
}

func sendCmdToWm(cmd string) {
	if socket == nil {
		return
	}

	var dataToSend []byte

	upper := strings.ToUpper(cmd)
	switch upper {
	case "THEME-RELOAD":
		dataToSend = []byte(upper)
	default:
		fmt.Printf("action '%s' unrecognized; ignoring\n", cmd)
		return
	}

	if _, err := socket.Write(dataToSend); err != nil {
		fmt.Printf("error sending '%s' to dwm unix socket: %s\n", dataToSend, err)
	} else {
		fmt.Printf("sent %s to WM\n", dataToSend)
	}
}

func initiateWmCommunication() {
	for {
		if socket, err = net.Dial("unix", config.WmSocket); err == nil {
			// Success, let's leave the loop.
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer socket.Close()

	buffer := make([]byte, 128)
	for {
		n, err := socket.Read(buffer[:])
		if err == nil {
			parseReceivedData(buffer[0:n])
		}
	}
}

func triggerWmReload() {
	sendCmdToWm("THEME-RELOAD")
	// Triggering WM to update its status bar and possibly theme.
	exec.Command("xsetroot", "-name", "").Run()
}

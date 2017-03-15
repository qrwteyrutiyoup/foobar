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
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	// The default length to adjust strings in the function
	// adjustStringWidth.
	adjustedWidthLen = 6
)

type screen struct {
	width  int
	height int
}

var (
	monitors []screen
)

// adjustStringWidth tries to adjust the width of the received
// string by left-trimming it from whitespaces.
func adjustStringWidth(s string, width int) string {
	s = strings.TrimLeft(s, " ")
	if len(s) < width {
		diff := width - len(s)
		for i := 0; i < diff; i++ {
			s += " "
		}
	}
	return s
}

// FormatBytes received a value in bytes and converts it to the
// largest unit so that its value >= 1. It also adjusts the string
// width to be adjustedWidthLen.
func formatBytes(b int) string {
	kb := float32(b) / 1000.0
	if kb < 1 {
		return adjustStringWidth(fmt.Sprintf("%4dB", b), adjustedWidthLen)
	}
	mb := kb / 1000.0
	if mb < 1 {
		return adjustStringWidth(fmt.Sprintf("%4.1fK", kb), adjustedWidthLen)
	}
	gb := mb / 1000.0
	if gb < 1 {
		return adjustStringWidth(fmt.Sprintf("%4.1fM", mb), adjustedWidthLen)
	}
	tb := gb / 1000.0
	if tb < 1 {
		return adjustStringWidth(fmt.Sprintf("%4.1fG", gb), adjustedWidthLen)
	}

	return adjustStringWidth(fmt.Sprintf("%4.1fT", tb), adjustedWidthLen)
}

// ProgressBar draws a three-icon string progress bar based on the
// value it receives and the list of icons.
func progressBar(value int) string {
	switch {
	case value < 10:
		return fmt.Sprintf("%s%s%s", icons["bar-left-0"], icons["bar-middle-0"], icons["bar-right-0"])
	case value < 20:
		return fmt.Sprintf("%s%s%s", icons["bar-left-1"], icons["bar-middle-0"], icons["bar-right-0"])
	case value < 30:
		return fmt.Sprintf("%s%s%s", icons["bar-left-2"], icons["bar-middle-0"], icons["bar-right-0"])
	case value < 40:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-0"], icons["bar-right-0"])
	case value < 50:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-1"], icons["bar-right-0"])
	case value < 60:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-2"], icons["bar-right-0"])
	case value < 70:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-3"], icons["bar-right-0"])
	case value < 80:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-4"], icons["bar-right-0"])
	case value < 90:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-4"], icons["bar-right-1"])
	case value < 100:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-4"], icons["bar-right-2"])
	default:
		return fmt.Sprintf("%s%s%s", icons["bar-left-3"], icons["bar-middle-4"], icons["bar-right-3"])
	}
}

// GetScreensInfo returns the list of connected monitors.
func getScreensInfo() {
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		fmt.Println("error getting screens: ", err)
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var line, resolution string
	var w, h int
	for scanner.Scan() {
		line = scanner.Text()
		if strings.Contains(line, " connected ") {
			if strings.Contains(line, "primary") {
				resolution = strings.Split(line, " ")[3]
			} else {
				resolution = strings.Split(line, " ")[2]
			}
			res := strings.Split(strings.Split(resolution, "+")[0], "x")
			w, err = strconv.Atoi(res[0])
			h, err = strconv.Atoi(res[1])

			monitors = append(monitors, screen{width: w, height: h})
		}
	}
	fmt.Println("Detected screens: ", monitors)
}

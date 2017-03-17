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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type colorInfo struct {
	SidebarsBg string
	SidebarsFg string
	Urgent     string
	Key        string
	Value      string
	Bg         string
}

type dzenInfo struct {
	name   string
	cmd    *exec.Cmd
	args   []string
	stdin  io.WriteCloser
	hidden bool
}

type popupConfig struct {
	Info    string
	Clock   string
	Weather string
	User    string
}

type barConfig struct {
	Height       int
	LeftBarWidth int
	Contiguous   string
	Position     string
}

type info struct {
	icon      string
	key       string
	value     string
	format    *string
	formatted string
	length    int
}

var (
	formatDefault string
	formatUrgent  string

	mainbarWidth  = 500
	leftBarWidth  = 0
	barHeight     = 15
	contiguousBar = true
	isTopBar      = true

	dzenMainbar []dzenInfo
	dzenLeftbar []dzenInfo

	err      error
	username string
)

const (
	// Yeah, I am wondering about this as well.
	barWidthMagic = 7.5
)

func initDzenBars() {
	dzenLeftbar = make([]dzenInfo, len(monitors))
	dzenMainbar = make([]dzenInfo, len(monitors))

}

func loadDzenColorFormats() {
	formatDefault = fmt.Sprintf("^fg(%s)%%s ^fg(%s)%%s", config.Colors.Key, config.Colors.Value)
	formatUrgent = fmt.Sprintf("^fg(%s)%%s %%s", config.Colors.Urgent)
}

func barWidthFromKey(key string) int {
	w := 0
	started := false
	for i := range keys {
		if keys[i] == key {
			started = true
		}

		if started {
			w += data[keys[i]].length
		}

	}
	w += len(username) + 5

	ret := int(float32(w) * barWidthMagic)
	return ret
}

func leftBarContent(screen int) string {
	if len(config.Popups.Info) > 0 {
		return fmt.Sprintf("^ca(1,%s %d %d %d)^fg(%s)^bg(%s)  info^fg(%s)^bg(%s)  ^ca()\n", config.Popups.Info, (screen + 1), monitors[screen].width, monitors[screen].height, config.Colors.SidebarsFg, config.Colors.SidebarsBg, config.Colors.SidebarsBg, config.Colors.Bg)
	}
	return fmt.Sprintf("^fg(%s)^bg(%s)  info^fg(%s)^bg(%s)  \n", config.Colors.SidebarsFg, config.Colors.SidebarsBg, config.Colors.SidebarsBg, config.Colors.Bg)
}

func statusBarLen() int {
	bar := ""
	var key string

	// Regular bar without any color formatting.
	for i := range keys {
		key = keys[i]
		if collected, ok := data[key]; ok {
			bar = fmt.Sprintf("%s %s %s", bar, collected.icon, collected.value)
		}
	}

	return len(bar)
}

func statusBar(screen int) string {
	bar := ""
	var key string

	for i := range keys {
		key = keys[i]
		if collected, ok := data[key]; ok {
			switch {
			case key == "clock" && len(config.Popups.Clock) > 0:
				barWidth := barWidthFromKey(key)
				bar = fmt.Sprintf("%s ^ca(1,%s %d %d %d %d)%s^ca()", bar, config.Popups.Clock, (screen + 1), monitors[screen].width, monitors[screen].height, barWidth, collected.formatted)
			case key == "weather" && len(config.Popups.Weather) > 0:
				barWidth := barWidthFromKey(key)
				bar = fmt.Sprintf("%s ^ca(1,%s %d %d %d %d)%s^ca()", bar, config.Popups.Weather, (screen + 1), monitors[screen].width, monitors[screen].height, barWidth, collected.formatted)
			default:
				bar = fmt.Sprintf("%s %s", bar, collected.formatted)
			}
		}
	}
	if len(config.Popups.User) > 0 {
		bar = fmt.Sprintf("%s ^ca(1,dzen-popup-user %d %d %d)^fg(%s)^fg(%s)^bg(%s) %s ^ca()", bar, (screen + 1), monitors[screen].width, monitors[screen].height, config.Colors.SidebarsBg, config.Colors.SidebarsFg, config.Colors.SidebarsBg, username)
	} else {
		bar = fmt.Sprintf("%s ^fg(%s)^fg(%s)^bg(%s) %s ", bar, config.Colors.SidebarsBg, config.Colors.SidebarsFg, config.Colors.SidebarsBg, username)
	}

	resizeDzenMainBar()
	return bar
}

func updateStatusBar() {
	collectStats()
	for i := 0; i < len(dzenMainbar); i++ {
		io.WriteString(dzenMainbar[i].stdin, fmt.Sprintf("%s\n", statusBar(i)))
	}
}

func resizeDzenMainBar() {
	if contiguousBar {
		return
	}

	if newStatusbarLen := int(float32(statusBarLen()) * barWidthMagic); newStatusbarLen != mainbarWidth {
		mainbarWidth = newStatusbarLen
		drawDzenMainBar()
	}
}

func reloadStatusBar() {
	collectVolume("volume")
	collectBrightness("brightness")

	for i := 0; i < len(monitors); i++ {
		io.WriteString(dzenMainbar[i].stdin, fmt.Sprintf("%s\n", statusBar(i)))
	}
}

func execDzen(args []string) (dzen *exec.Cmd, stdin io.WriteCloser, err error) {
	dzen = exec.Command("dzen2", args...)
	stdin, err = dzen.StdinPipe()
	if err != nil {
		return
	}

	dzen.Stdout = os.Stdout
	dzen.Stderr = os.Stderr

	err = dzen.Start()
	return
}

func closeDzenByMonitor(monitor int) {
	closeDzenBar(&dzenLeftbar[monitor])
	closeDzenBar(&dzenMainbar[monitor])
}

func closeDzenBar(dzen *dzenInfo) {
	if dzen != nil && dzen.stdin != nil && dzen.cmd != nil {
		dzen.stdin.Close()
		dzen.hidden = true
		dzen.cmd.Wait()
	}
}

func closeDzen(dzen []dzenInfo, delay bool) {
	if delay {
		// Delay for 1 second the closing of the old stdin.
		time.Sleep(1 * time.Second)
	}

	for i := 0; i < len(dzen); i++ {
		closeDzenBar(&dzen[i])
	}
}

func drawDzenMainBarByMonitor(monitor int) (dzenInfo, error) {
	if monitor < 0 || monitor >= len(monitors) {
		return dzenInfo{}, errors.New("Monitor index incorrect")
	}

	width := monitors[monitor].width - leftBarWidth
	x := leftBarWidth

	if !contiguousBar {
		x = monitors[monitor].width - mainbarWidth - 1
		width = mainbarWidth
	}

	y := 0
	if !isTopBar {
		y = monitors[monitor].height - barHeight
	}

	dzenArgs := []string{"-xs", fmt.Sprintf("%d", (monitor + 1)), "-ta", "r", "-fn", config.Font, "-x", fmt.Sprintf("%d", x), "-y", fmt.Sprintf("%d", y), "-w", fmt.Sprintf("%d", width), "-h", fmt.Sprintf("%d", barHeight), "-bg", config.Colors.Bg, "-fg", config.Colors.Key, "-e", "button2=;"}

	cmd, dzenStdin, _ := execDzen(dzenArgs)
	io.WriteString(dzenStdin, fmt.Sprintf("%s\n", statusBar(monitor)))

	dzenBar := dzenInfo{name: fmt.Sprintf("main bar, monitor %d", monitor), cmd: cmd, args: dzenArgs, stdin: dzenStdin, hidden: false}
	dzenMainbar[monitor] = dzenBar
	return dzenBar, nil
}

func drawDzenMainBar() {
	nscreens := len(monitors)

	var oldBar dzenInfo

	for i := 0; i < nscreens; i++ {
		oldBar = dzenMainbar[i]
		if _, err := drawDzenMainBarByMonitor(i); err == nil {
			go closeDzenBar(&oldBar)
		}
	}

}

func drawDzenLeftBarByMonitor(monitor int) (dzenInfo, error) {
	if monitor < 0 || monitor >= len(monitors) {
		return dzenInfo{}, errors.New("Monitor index incorrect")
	}

	y := 0
	if !isTopBar {
		y = monitors[monitor].height - barHeight
	}

	dzenArgs := []string{"-xs", fmt.Sprintf("%d", (monitor + 1)), "-ta", "l", "-fn", config.Font, "-w", fmt.Sprintf("%d", leftBarWidth), "-h", fmt.Sprintf("%d", barHeight), "-x", "0", "-y", fmt.Sprintf("%d", y), "-bg", config.Colors.Bg, "-fg", config.Colors.Key, "-e", "button2=;"}

	cmd, dzenStdin, _ := execDzen(dzenArgs)
	io.WriteString(dzenStdin, leftBarContent(monitor))

	dzenBar := dzenInfo{name: fmt.Sprintf("left bar, monitor %d", monitor), cmd: cmd, args: dzenArgs, stdin: dzenStdin, hidden: false}
	dzenLeftbar[monitor] = dzenBar
	return dzenBar, nil
}

func drawDzenLeftBar() {
	nscreens := len(monitors)

	var oldBar dzenInfo

	for i := 0; i < nscreens; i++ {
		oldBar = dzenLeftbar[i]
		if _, err := drawDzenLeftBarByMonitor(i); err == nil {
			go closeDzenBar(&oldBar)
		}
	}
}

func drawDzenBars() {
	drawDzenLeftBar()
	drawDzenMainBar()
	resizeDzenMainBar()
}

func drawDzenByMonitor(monitor int) {
	drawDzenLeftBarByMonitor(monitor)
	drawDzenMainBarByMonitor(monitor)
}

func updateDzenConfig() {
	contiguousBar = true
	if len(config.Bar.Contiguous) > 0 && config.Bar.Contiguous != "yes" {
		contiguousBar = false
	}

	barHeight = 15
	if config.Bar.Height > 0 {
		barHeight = config.Bar.Height
	}

	leftBarWidth = 0
	if config.Bar.LeftBarWidth > 0 {
		leftBarWidth = config.Bar.LeftBarWidth
	}

	isTopBar = true
	if config.Bar.Position == "bottom" {
		isTopBar = false
	}
}

func toggleBars(monitor int) {
	hidden := dzenLeftbar[monitor].hidden

	// close/drawDzenByMonitor take care of their hidden status.
	if !hidden {
		closeDzenByMonitor(monitor)
	} else {
		drawDzenByMonitor(monitor)
	}
}

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
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	app     = "foobar"
	version = "0.1, March 15 2017"
	author  = "Sergio Correia <sergio@correia.cc>"
)

func main() {
	fmt.Printf("%s v%s\nCopyright (C) 2017 by %s\n", app, version, author)

	configFile = defaultConfigFile()
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		usage(configFile)
	}

	// Information on the number of monitors.
	getScreensInfo()

	initDzenBars()

	data = make(map[string]info)
	loadConfig()

	// Bidirectional communication with WM via Unix domain socket.
	go initiateWmCommunication()

	network = networkInfo{validDevice: isValidNetDevice(), rxOld: 0, rxUpdateTime: 1, txOld: 0, txUpdateTime: 1}

	signalChan := make(chan os.Signal, 1)
	go func() {
		for {
			s := <-signalChan
			switch s {
			case syscall.SIGHUP:
				fmt.Println("Reloading config...")
				loadConfig()

				// Reformatting info with possibly a new color theme.
				updateFormatting()

				triggerWmReload()

				collectStats()
				drawDzenBars()

				updateStatusBar()
			case syscall.SIGUSR1:
				// Trigger a reload of the bar, to update info
				// like the volume or brightness indicator.
				reloadStatusBar()
			default:
				fmt.Println(s)
			}
		}
	}()

	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGUSR1)

	drawDzenBars()

	for {
		updateStatusBar()

		// Sleep until the beginning of next second.
		var now = time.Now()
		time.Sleep(now.Truncate(time.Second).Add(time.Second).Sub(now))
	}
}

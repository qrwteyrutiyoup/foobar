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
	"encoding/json"
	"fmt"
	"os"
	"os/user"
)

type wmIcon struct {
	Name string
	Icon string
}

type wmConfig struct {
	SoundDevice      string
	NetworkInterface string
	Font             string
	WmSocket         string
	Icons            []wmIcon
	Weather          weatherInfo
	Colors           colorInfo
	Bar              barConfig
	Popups           popupConfig
}

var (
	configFile string
	config     wmConfig
	icons      map[string]string
)

func usage(filename string) {
	fmt.Printf("Config file '%s' does not seem to exist. Please double check.\n", filename)
	fmt.Printf("Usage: %s [config file]\n\n", app)
	fmt.Printf("If no config file is specified, %s will try to use '$XDG_CONFIG_HOME/foobar/foobar.cfg', if $XDG_CONFIG_HOME is set, or '~/.config/foobar/foobar.cfg', otherwise.\n", app)
	os.Exit(1)
}

func configDirectory() string {
	// From XDG Base Directory Specification
	//
	// $XDG_CONFIG_HOME defines the base directory relative to which user specific
	// configuration files should be stored. If $XDG_CONFIG_HOME is either not set
	// or empty, a default equal to $HOME/.config should be used.
	baseDir := os.Getenv("XDG_CONFIG_HOME")
	if len(baseDir) == 0 {
		currentUser, _ := user.Current()
		baseDir = fmt.Sprintf("%s/.config", currentUser.HomeDir)
	}
	return baseDir
}

func defaultConfigFile() string {
	return fmt.Sprintf("%s/%s/foobar.cfg", configDirectory(), app)
}

func loadConfig() {
	jsonConfig, err := os.Open(configFile)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read config file '%s': %s\nexiting...\n", configFile, err)
		os.Exit(1)
	}
	defer jsonConfig.Close()

	config = wmConfig{}
	jsonParser := json.NewDecoder(jsonConfig)

	if err = jsonParser.Decode(&config); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file '%s': %s\nexiting...\n", configFile, err)
		os.Exit(2)
	}

	updateDzenConfig()

	icons = make(map[string]string)
	loadDzenColorFormats()

	for i := range config.Icons {
		icons[config.Icons[i].Name] = config.Icons[i].Icon
	}

	validSoundDevice = isValidSoundDevice()

	if validWeatherAPIKey = isValidWeatherAPIKey(config.Weather.APIKey); validWeatherAPIKey {
		os.Setenv("OWM_API_KEY", config.Weather.APIKey)
	} else {
		os.Unsetenv("OWM_API_KEY")
	}

	username = os.Getenv("USER")
}

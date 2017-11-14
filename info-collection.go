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
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type networkInfo struct {
	validDevice  bool
	rxOld        int
	rxUpdateTime int
	txOld        int
	txUpdateTime int
}

var (
	data map[string]info

	network networkInfo
	keys    = []string{"clock", "rx", "tx", "volume", "battery", "brightness", "cpu", "ram"}

	validSoundDevice = false
	cores            = runtime.NumCPU()
)

func isValidNetDevice() bool {
	out, err := exec.Command("ip", "addr", "show", config.NetworkInterface).Output()
	if err != nil || len(out) == 0 {
		removeKey("rx")
		removeKey("tx")
		fmt.Printf("Network device '%s' is not valid; please recheck the config file\n", config.NetworkInterface)
		return false
	}

	return true
}

func isValidSoundDevice() bool {
	out, err := exec.Command("pactl", "list", "sinks", "short").Output()
	if err != nil || len(out) == 0 {
		fmt.Printf("Sound device '%s' is not valid; please recheck the config file\n", config.SoundDevice)
		removeKey("volume")
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		deviceID := strings.Split(scanner.Text(), "\t")[0]
		if deviceID == config.SoundDevice {
			return true
		}
	}
	removeKey("volume")
	fmt.Printf("Sound device '%s' is not valid; please recheck the config file\n", config.SoundDevice)
	return false
}

func formatData(key, value, icon string, format *string) {
	data[key] = info{icon: icon, key: key, value: value, format: format, formatted: fmt.Sprintf(*format, icon, value), length: len(value) + 2}
}

func updateFormatting() {
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		if current, ok := data[key]; ok {
			formatData(key, current.value, current.icon, current.format)
		}
	}
}

func removeKey(key string) {
	if _, ok := data[key]; ok {
		delete(data, key)
	}
}

func collectTime(key string) {
	t := time.Now()

	formatData(key, t.Format("15:04:05"), icons[key], &formatDefault)
}

func collectNetwork(rxkey, txkey string) {
	if !network.validDevice {
		return
	}

	// Based in https://github.com/schachmat/gods/blob/master/gods.go
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		removeKey(rxkey)
		removeKey(txkey)
		return
	}
	defer file.Close()

	var void = 0 // target for unused values
	var dev, rx, tx, rxNow, txNow = "", 0, 0, 0, 0
	var scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		_, err = fmt.Sscanf(scanner.Text(), "%s %d %d %d %d %d %d %d %d %d",
			&dev, &rx, &void, &void, &void, &void, &void, &void, &void, &tx)

		if dev[:len(dev)-1] == config.NetworkInterface {
			rxNow += rx
			txNow += tx
		}
	}

	defer func() { network.rxOld, network.txOld = rxNow, txNow }()
	rxdata := fmt.Sprintf("%s", formatBytes(rxNow-network.rxOld))
	txdata := fmt.Sprintf("%s", formatBytes(txNow-network.txOld))

	network.rxUpdateTime--
	if network.rxUpdateTime <= 0 {
		formatData(rxkey, rxdata, icons[rxkey], &formatDefault)
		network.rxUpdateTime = rand.Intn(2) + 1
	}

	network.txUpdateTime--
	if network.txUpdateTime <= 0 {
		formatData(txkey, txdata, icons[txkey], &formatDefault)
		network.txUpdateTime = rand.Intn(2) + 1
	}
}

func collectBrightness(key string) {
	actual, err := ioutil.ReadFile("/sys/class/backlight/intel_backlight/actual_brightness")
	if err != nil {
		removeKey(key)
		return
	}

	max, err := ioutil.ReadFile("/sys/class/backlight/intel_backlight/max_brightness")
	if err != nil {
		removeKey(key)
		return
	}

	var actualBr, maxBr int
	_, err = fmt.Sscanf(string(actual), "%d", &actualBr)
	if err != nil {
		removeKey(key)
		return
	}
	_, err = fmt.Sscanf(string(max), "%d", &maxBr)
	if err != nil {
		removeKey(key)
		return
	}

	cur := 100 * actualBr / maxBr
	formatData(key, progressBar(cur), icons[key], &formatDefault)
}

func collectPower(key string) {
	out, err := exec.Command("acpi", "-b").Output()
	if err != nil || len(out) == 0 {
		removeKey(key)
		return
	}

	output := string(out)

	split := strings.Split(output, " ")
	charge := split[3][:len(split[3])-1]

	var value int
	var icon string
	_, err = fmt.Sscanf(charge, "%d", &value)
	if err != nil {
		removeKey(key)
		return
	}

	var iconName string
	switch {
	case value > 75:
		iconName = "battery_full"
	case value > 50:
		iconName = "battery_three_quarters"
	case value > 25:
		iconName = "battery_half"
	case value > 10:
		iconName = "battery_quarter"
	default:
		iconName = "battery_empty"
	}

	if strings.Contains(output, "Charging") || strings.Contains(output, "will never fully discharge") {
		iconName = fmt.Sprintf("%s_power", iconName)
	}
	icon = icons[iconName]

	format := &formatDefault
	if value <= 10 {
		format = &formatUrgent
	}

	formatData(key, progressBar(value), icon, format)
}

func collectRAM(key string) {
	// from: https://github.com/schachmat/gods/blob/master/gods.go
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		removeKey(key)
		return
	}
	defer file.Close()

	// done must equal the flag combination (0001 | 0010 | 0100 | 1000) = 15
	var total, used, done = 0, 0, 0
	for info := bufio.NewScanner(file); done != 15 && info.Scan(); {
		var prop, val = "", 0
		if _, err = fmt.Sscanf(info.Text(), "%s %d", &prop, &val); err != nil {
			removeKey(key)
			return
		}

		switch prop {
		case "MemTotal:":
			total = val
			used += val
			done |= 1
		case "MemFree:":
			used -= val
			done |= 2
		case "Buffers:":
			used -= val
			done |= 4
		case "Cached:":
			used -= val
			done |= 8
		}
	}

	ram := used * 100 / total
	formatData(key, progressBar(ram), icons[key], &formatDefault)
}

func collectCPU(key string) {
	// from: https://github.com/schachmat/gods/blob/master/gods.go
	var load float32
	loadavg, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		removeKey(key)
		return
	}

	_, err = fmt.Sscanf(string(loadavg), "%f", &load)
	if err != nil {
		removeKey(key)
		return
	}

	cpu := int(load * 100.0 / float32(cores))
	formatData(key, progressBar(cpu), icons[key], &formatDefault)
}

func collectVolume(key string) {
	if !validSoundDevice {
		return
	}

	deviceID, err := strconv.Atoi(config.SoundDevice)
	if err != nil {
		removeKey(key)
		return
	}

	out, err := exec.Command("pactl", "list", "sinks").Output()
	if err != nil {
		removeKey(key)
		return
	}

	output := string(out)
	var trimmed string
	volumes := make([]string, deviceID+1, deviceID+1)
	muted := make([]bool, deviceID+1, deviceID+1)
	headphone := make([]bool, deviceID+1, deviceID+1)
	volumesIndex := 0
	muteIndex := 0
	headphoneIndex := 0
	reInsideWhitespace := regexp.MustCompile(`[\s\p{Zs}]{2,}`)

	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		trimmed = reInsideWhitespace.ReplaceAllString(strings.TrimLeft(scanner.Text(), " \t\r\n"), " ")
		if strings.HasPrefix(trimmed, "Mute: ") {
			// Mute: no
			muted[muteIndex] = (strings.Split(trimmed, " ")[1] == "yes")
			muteIndex++
		}
		if strings.HasPrefix(trimmed, "Volume: ") {
			// Volume: front-left: 43055 /  66% / -10.95 dB,   front-right: 43055 /  66% / -10.95 dB
			volumes[volumesIndex] = strings.Split(trimmed, " ")[4]
			volumesIndex++
		}
		if strings.HasPrefix(trimmed, "Active Port: ") {
			headphone[headphoneIndex] = strings.Contains(strings.Split(trimmed, " ")[2], "headphones")
		}
	}

	volume, err := strconv.Atoi(volumes[deviceID][:len(volumes[deviceID])-1])
	if err != nil {
		removeKey(key)
		return
	}

	var icon string

	format := &formatDefault
	if muted[deviceID] {
		format = &formatUrgent
		if headphone[deviceID] {
			icon = icons["headphone_mute"]
		} else {
			if volume > 40 {
				icon = icons["volume_loud_mute"]
			} else {
				icon = icons["volume_low_mute"]
			}
		}
	} else {
		if headphone[deviceID] {
			icon = icons["headphone"]
		} else {
			if volume > 40 {
				icon = icons["volume_loud"]
			} else {
				icon = icons["volume_low"]
			}
		}
	}

	formatData(key, progressBar(volume), icon, format)
}

func collectStats() {
	collectVolume("volume")
	collectTime("clock")
	collectRAM("ram")
	collectCPU("cpu")
	collectNetwork("rx", "tx")
	collectPower("battery")
	collectBrightness("brightness")
}

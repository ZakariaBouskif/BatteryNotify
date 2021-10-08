package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gopkg.in/toast.v1"
)

/*
Battery Status from Win32_Battery Class in Windows OS
for more infos : https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/win32-battery
*/
var BatteryStatus = map[int]string{
	1:  "discharging",
	2:  "Plugged in",
	3:  "Fully Charged",
	4:  "low",
	5:  "critical",
	6:  "Charging",
	7:  "Charging and High",
	8:  "Charging and Low",
	9:  "Charging and Critical",
	10: "Undefined",
	11: "Partially Charged",
}

var lastNotify time.Time

type Configuration struct {
	MaxLevel  int
	MinLevel  int
	Timer     int
	Remainder int
}

func loadConfiguration() (*Configuration, error) {
	configFile, err := os.Open("conf.json")

	if err != nil {
		return nil, errors.New("error while openning config file")
	}

	defer configFile.Close()

	jsonData, _ := ioutil.ReadAll(configFile)
	configData := &Configuration{}
	json.Unmarshal(jsonData, &configData)

	if configData.MaxLevel <= 0 || configData.MinLevel <= 0 {
		return nil, errors.New("error MaxLevel or MinLevel cannout be 0 or less")
	} else if configData.MaxLevel <= configData.MinLevel {
		return nil, errors.New("error MaxLevel should be geather than MinLevel")
	}

	if configData.Timer < 5 || configData.Remainder < 15 {
		return nil, errors.New("error expected values : Timer >= 5 and Remainder >= 15")
	}

	return configData, nil
}

// Get the index of battery status
func getBatteryStatus() int {
	out, _ := exec.Command("WMIC", "PATH", "Win32_Battery", "Get", "BatteryStatus").Output()
	output := string(out)
	v := strings.Split(output, "\r\r\n")
	i, _ := strconv.Atoi(strings.Replace(v[1], " ", "", -1))

	return i
}

// Get the value of battery level e.g : 80 mean 80%
func getBatteryLevel() int {
	out, _ := exec.Command("WMIC", "PATH", "Win32_Battery", "Get", "EstimatedChargeRemaining").Output()

	output := string(out)
	v := strings.Split(output, "\r\r\n")
	i, _ := strconv.Atoi(strings.Replace(v[1], " ", "", -1))

	return i
}

func main() {

	fmt.Println("Battery Notify is running...")

	// loading configuaration from "config.json file"
	config, err := loadConfiguration()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	t := time.NewTicker(time.Duration(config.Timer) * time.Second) // Callback every specific time

	for range t.C {
		batteryStatus := getBatteryStatus()
		batteryInt := getBatteryLevel()

		lastNotifyTime := int(time.Since(lastNotify)) / 1000000000

		// reminde user every "config.Remainder" seconds
		if lastNotifyTime > config.Remainder {
			/*
				Case when the battery is plugged in and the its level >= max
			*/
			if batteryStatus == 2 && batteryInt >= config.MaxLevel {
				Alert("Battery Status: "+BatteryStatus[batteryStatus], "You should unplugging your charger", "full")
			}

			/*
				Case when the battery is unpluggedand and its level <= min
			*/
			if batteryStatus == 1 && batteryInt <= config.MinLevel {
				Alert("Battery Status: "+BatteryStatus[batteryStatus], "You should plugging your charger", "low")
			}

			lastNotify = time.Now()
		}
	}
}

// Alert displays a desktop notification and plays a default system sound.
func Alert(title, message, appIcon string) {

	dir, _ := os.Getwd()

	notification := toast.Notification{
		AppID:   "Battery Notify",
		Title:   title,
		Message: message,
		Icon:    dir + "\\" + appIcon + ".png", // This file must exist (remove this line if it doesn't)
		Actions: []toast.Action{},
	}
	err := notification.Push()
	if err != nil {
		log.Fatalln(err)
	}
}

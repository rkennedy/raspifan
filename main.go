package main

import (
	"bufio"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/okzk/sdnotify"
	"github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/sys/unix"
	"gopkg.in/yaml.v3"
)

type Config struct {
	MaximumTemperature float64 // degrees C
	TargetTemperature  float64
	PollingInterval    time.Duration
	Pin                uint
}

func (c *Config) Log(msg string) {
	slog.Info(msg,
		slog.Float64("max", c.MaximumTemperature),
		slog.Float64("target", c.TargetTemperature),
		slog.Duration("interval", c.PollingInterval),
		slog.Uint64("pin", uint64(c.Pin)),
	)
}

func loadConfig() (Config, error) {
	result := Config{
		MaximumTemperature: 80.0,
		TargetTemperature:  60.0,
		PollingInterval:    2 * time.Minute,
		Pin:                14,
	}

	config, err := os.Open("/etc/raspifan.conf")
	if errors.Is(err, fs.ErrNotExist) {
		result.Log("No config file; using defaults.")
		return result, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer config.Close()

	decoder := yaml.NewDecoder(config)
	err = decoder.Decode(&result)
	if err != nil {
		return Config{}, err
	}
	result.Log("Loaded config.")
	return result, err
}

func fanTemp() (float64, error) {
	temp, err := os.Open("/sys/devices/virtual/thermal/thermal_zone0/temp")
	if err != nil {
		slog.Error("Unable to open temperature file.",
			slog.String("error", err.Error()),
		)
		return 0.0, err
	}
	defer temp.Close()
	scanner := bufio.NewScanner(temp)
	if !scanner.Scan() {
		slog.Error("Unable to read temperature from file.",
			slog.String("error", err.Error()),
		)
		return 0.0, scanner.Err()
	}
	result, err := strconv.Atoi(scanner.Text())
	if err != nil {
		slog.Error("Cannot convert temperature.",
			slog.String("input", scanner.Text()),
			slog.String("error", err.Error()),
		)
		return 0.0, err
	}
	return float64(result) / 1000.0, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("Cannot load configuration.",
			slog.String("error", err.Error()),
		)
		sdnotify.Errno(int(unix.EINVAL))
		return
	}

	var watchdog <-chan time.Time
	if watchdogFreq, ok := os.LookupEnv("WATCHDOG_USEC"); ok {
		freq, err := strconv.Atoi(watchdogFreq)
		if err != nil {
			slog.Error("Invalid WATCHDOG_USEC.",
				slog.String("value", watchdogFreq),
				slog.String("error", err.Error()),
			)
			sdnotify.Errno(int(unix.EINVAL))
			return
		}
		freqD := time.Duration(freq) * time.Microsecond / 2
		slog.Info("Will send watchdog notifications.",
			slog.Duration("frequency", freqD),
		)
		watchdog = time.Tick(freqD)
	}

	err = rpio.Open()
	if err != nil {
		slog.Error("Cannot open I/O pin.",
			slog.String("error", err.Error()),
		)
		return
	}
	defer rpio.Close()

	fanPin := rpio.Pin(config.Pin)
	fanPin.Output()
	if fanPin.Read() == rpio.High {
		slog.Info("Fan is on.")
	} else {
		slog.Info("Fan is off.")
	}

	sighup := make(chan os.Signal)
	signal.Notify(sighup, unix.SIGHUP)
	defer signal.Stop(sighup)
	sigterm := make(chan os.Signal)
	signal.Notify(sigterm, unix.SIGTERM, unix.SIGINT)
	defer signal.Stop(sigterm)

	pollTicker := time.NewTicker(config.PollingInterval)
	defer pollTicker.Stop()

	sdnotify.Ready()
	checkTemperature(&fanPin, config.MaximumTemperature, config.TargetTemperature)
	for {
		select {
		case <-sigterm:
			sdnotify.Stopping()
			return
		case <-sighup:
			sdnotify.Reloading()
			newConfig, err := loadConfig()
			if err != nil {
				sdnotify.Status(err.Error())
			} else {
				if config.PollingInterval != newConfig.PollingInterval {
					pollTicker.Reset(newConfig.PollingInterval)
				}
				if config.Pin != newConfig.Pin {
					fanPin = rpio.Pin(newConfig.Pin)
					fanPin.Output()
				}
				if config.MaximumTemperature != newConfig.MaximumTemperature || config.TargetTemperature != newConfig.TargetTemperature {
					checkTemperature(&fanPin, newConfig.MaximumTemperature, newConfig.TargetTemperature)
				}
				config = newConfig
			}
			sdnotify.Ready()

		case <-pollTicker.C:
			checkTemperature(&fanPin, config.MaximumTemperature, config.TargetTemperature)
		case <-watchdog:
			sdnotify.Watchdog()
		}
	}
}

func checkTemperature(fanPin *rpio.Pin, maxTemp, targetTemp float64) {
	temp, err := fanTemp()
	if err != nil {
		sdnotify.Status("cannot probe")
		return
	}
	fan := fanPin.Read()
	slog.Info("Polled.",
		slog.Float64("temperature", temp),
		slog.Int("pin", int(fan)),
	)
	if temp >= maxTemp && fan == rpio.Low {
		// Turn the fan on.
		fanPin.High()
		sdnotify.Status("fan on")
	} else if temp <= targetTemp && fan == rpio.High {
		// Turn the fan off.
		fanPin.Low()
		sdnotify.Status("fan off")
	}
}

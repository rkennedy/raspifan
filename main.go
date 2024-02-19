package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/okzk/sdnotify"
	"github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/sys/unix"
)

func main() {
	if len(os.Args) > 0 && os.Args[0] == "install" {
		err := install()
		if err != nil {
			slog.Error("Cannot install systemd file.",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		return
	}

	config, err := loadConfig()
	if err != nil {
		slog.Error("Cannot load configuration.",
			slog.String("error", err.Error()),
		)
		sdnotify.Errno(int(unix.EINVAL))
		return
	}

	watchdog, err := getWatchdogNotifications()
	if err != nil {
		// Function already did logging.
		return
	}

	err = rpio.Open()
	if err != nil {
		slog.Error("Cannot open I/O pin.",
			slog.String("error", err.Error()),
		)
		sdnotify.Errno(int(unix.EINVAL))
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
	temp, err := readTemp()
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

func getWatchdogNotifications() (<-chan time.Time, error) {
	// Start off nil. If we're running under systemd, with a watchdog
	// interval set, then set up this timer. It's safe to use a nil channel
	// in `select` below.
	watchdogFreq, ok := os.LookupEnv("WATCHDOG_USEC")
	if !ok {
		slog.Info("No watchdog notifications.")
		return nil, nil
	}
	freqUsec, err := strconv.Atoi(watchdogFreq)
	if err != nil {
		slog.Error("Invalid WATCHDOG_USEC.",
			slog.String("value", watchdogFreq),
			slog.String("error", err.Error()),
		)
		sdnotify.Errno(int(unix.EINVAL))
		return nil, err
	}
	// We're supposed to notify at half the watchdog interval.
	freq := time.Duration(freqUsec) * time.Microsecond / 2
	slog.Info("Will send watchdog notifications.",
		slog.Duration("frequency", freq),
	)
	return time.Tick(freq), nil
}

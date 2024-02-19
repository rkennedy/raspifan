package main

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"time"

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

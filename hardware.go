package main

import (
	"bufio"
	"log/slog"
	"os"
	"strconv"
)

const temperaturePath = "/sys/devices/virtual/thermal/thermal_zone0/temp"

func readTemp() (float64, error) {
	temp, err := os.Open(temperaturePath)
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

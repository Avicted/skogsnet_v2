package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type Measurement struct {
	UnixTimestamp      int64
	TemperatureCelsius float64
	HumidityPercentage float64
}

func deserializeData(data string) (Measurement, error) {
	var measurement Measurement
	type raw struct {
		TemperatureCelsius float64 `json:"temperature_celcius"`
		HumidityPercentage float64 `json:"humidity"`
	}
	var r raw
	if err := json.Unmarshal([]byte(data), &r); err != nil {
		return Measurement{}, fmt.Errorf("failed to deserialize data: %w", err)
	}
	measurement.TemperatureCelsius = r.TemperatureCelsius
	measurement.HumidityPercentage = r.HumidityPercentage
	measurement.UnixTimestamp = time.Now().UnixMilli()

	return measurement, nil
}

func printToConsole(measurement Measurement) {
	t := time.UnixMilli(measurement.UnixTimestamp)

	const (
		green  = "\033[32m"
		cyan   = "\033[36m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	fmt.Printf("%sMeasurement at %s%s\n", cyan, t.Format("2006-01-02 15:04:05"), reset)
	fmt.Printf("    %sTemperature:%s %s%.2f °C%s\n", green, reset, reset, measurement.TemperatureCelsius, reset)
	fmt.Printf("    %sHumidity:   %s %s%.2f %%%s\n", green, reset, reset, measurement.HumidityPercentage, reset)
}

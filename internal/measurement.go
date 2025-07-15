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

var deserializeData = deserializeDataImpl
var printToConsole = printToConsoleImpl

func deserializeDataImpl(data string) (Measurement, error) {
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

func printToConsoleImpl(measurement Measurement, weather *Weather) {
	t := time.UnixMilli(measurement.UnixTimestamp)

	const (
		green  = "\033[32m"
		cyan   = "\033[36m"
		yellow = "\033[33m"
		reset  = "\033[0m"
	)

	fmt.Printf("%sMeasurement at %s%s\n", cyan, t.Format("2006-01-02 15:04:05"), reset)
	fmt.Printf("    %sTemperature:        %s %s%.2f °C%s\n", green, reset, reset, measurement.TemperatureCelsius, reset)
	fmt.Printf("    %sHumidity:           %s %s%.2f %%%s\n", green, reset, reset, measurement.HumidityPercentage, reset)

	if weather != nil && len(weather.Weather) > 0 {
		fmt.Printf("\n")
		fmt.Printf("    %sWeather:            %s %s\n", green, reset, weather.Weather[0].Description)
		fmt.Printf("    %sOutside Temperature:%s %.2f °C\n", green, reset, weather.Main.Temp)
		fmt.Printf("    %sOutside Humidity:   %s %d%%\n", green, reset, weather.Main.Humidity)
		fmt.Printf("    %sWind Speed:         %s %.2f m/s\n", green, reset, weather.Wind.Speed)
		fmt.Printf("    %sWind Direction:     %s %d° %s\n", green, reset, weather.Wind.Deg, WindDirectionToCompass(weather.Wind.Deg))
		fmt.Printf("    %sCloud Cover:        %s %d%%\n", green, reset, weather.Clouds.All)
	}
}

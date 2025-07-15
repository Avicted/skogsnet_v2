package main

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestDeserializeData_ValidJSON(t *testing.T) {
	jsonStr := `{"temperature_celcius":22.5,"humidity":55.1}`
	m, err := deserializeData(jsonStr)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if m.TemperatureCelsius != 22.5 {
		t.Errorf("Expected TemperatureCelsius 22.5, got %v", m.TemperatureCelsius)
	}
	if m.HumidityPercentage != 55.1 {
		t.Errorf("Expected HumidityPercentage 55.1, got %v", m.HumidityPercentage)
	}
	if m.UnixTimestamp <= 0 {
		t.Errorf("Expected positive UnixTimestamp, got %v", m.UnixTimestamp)
	}
}

func TestDeserializeData_InvalidJSON(t *testing.T) {
	jsonStr := `{"temperature_celcius":"bad","humidity":55.1}`
	_, err := deserializeData(jsonStr)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestDeserializeData_MissingFields(t *testing.T) {
	jsonStr := `{"temperature_celcius":22.5}`
	m, err := deserializeData(jsonStr)
	if err != nil {
		t.Fatalf("Expected no error for missing humidity, got %v", err)
	}
	if m.HumidityPercentage != 0 {
		t.Errorf("Expected HumidityPercentage 0, got %v", m.HumidityPercentage)
	}
}

func TestDeserializeData_ExtraFields(t *testing.T) {
	jsonStr := `{"temperature_celcius":25.0,"humidity":60.0,"extra":123}`
	m, err := deserializeData(jsonStr)
	if err != nil {
		t.Fatalf("Expected no error for extra fields, got %v", err)
	}
	if m.TemperatureCelsius != 25.0 || m.HumidityPercentage != 60.0 {
		t.Errorf("Unexpected values: %v", m)
	}
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

func TestPrintToConsole_WithWeather(t *testing.T) {
	measurement := Measurement{
		UnixTimestamp:      1721049600000, // 2024-07-15 00:00:00 UTC
		TemperatureCelsius: 23.5,
		HumidityPercentage: 60.2,
	}
	weather := &Weather{
		Name: "Test City",
		Main: struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		}{
			Temp:     25.0,
			Humidity: 70,
		},
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		}{
			Speed: 5.5,
			Deg:   90,
		},
		Clouds: struct {
			All int `json:"all"`
		}{
			All: 80,
		},
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		}{
			{ID: 800, Main: "Clear", Description: "clear sky"},
		},
	}

	// Capture stdout
	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
		w.Close()
	}()
	go func() {
		io.Copy(&buf, r)
		r.Close()
	}()

	printToConsole(measurement, weather)
	time.Sleep(100 * time.Millisecond) // Ensure output is flushed

	output := buf.String()
	output = stripANSI(output)

	if !strings.Contains(output, "Measurement at") {
		t.Error("Expected output to contain 'Measurement at'")
	}
	if !strings.Contains(output, "Temperature:") {
		t.Error("Expected output to contain 'Temperature:'")
	}
	if !strings.Contains(output, "Humidity:") {
		t.Error("Expected output to contain 'Humidity:'")
	}
	if !strings.Contains(output, "Weather:") {
		t.Error("Expected output to contain 'Weather:'")
	}
	if !strings.Contains(output, "clear sky") {
		t.Error("Expected output to contain weather description")
	}
	if !strings.Contains(output, "Outside Temperature:") {
		t.Error("Expected output to contain 'Outside Temperature:'")
	}
	if !strings.Contains(output, "Wind Speed:") {
		t.Error("Expected output to contain 'Wind Speed:'")
	}
	if !strings.Contains(output, "Wind Direction:") {
		t.Error("Expected output to contain 'Wind Direction:'")
	}
	if !strings.Contains(output, "Cloud Cover:") {
		t.Error("Expected output to contain 'Cloud Cover:'")
	}
}

func TestPrintToConsole_NoWeather(t *testing.T) {
	measurement := Measurement{
		UnixTimestamp:      1752595488000,
		TemperatureCelsius: 21.1,
		HumidityPercentage: 44.2,
	}

	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = stdout
		w.Close()
	}()
	go func() {
		io.Copy(&buf, r)
		r.Close()
	}()

	printToConsole(measurement, nil)
	time.Sleep(100 * time.Millisecond) // Ensure output is flushed

	output := buf.String()
	output = stripANSI(output)

	if !strings.Contains(output, "Measurement at") {
		t.Error("Expected output to contain 'Measurement at'")
	}
	if !strings.Contains(output, "Temperature:         21.10 °C") {
		t.Error("Expected output to contain formatted temperature")
	}
	if !strings.Contains(output, "Humidity:            44.20 %") {
		t.Error("Expected output to contain formatted humidity")
	}
	if strings.Contains(output, "Weather:") {
		t.Error("Did not expect output to contain 'Weather:' when weather is nil")
	}
}

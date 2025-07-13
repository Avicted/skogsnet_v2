package main

import (
	"os"
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

func TestWriteToFile(t *testing.T) {
	tmpFile := "test_measurements.dat"
	defer os.Remove(tmpFile)

	file, err := openMeasurementFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open measurement file: %v", err)
	}
	defer file.Close()

	m := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 20.0,
		HumidityPercentage: 50.0,
	}
	if err := writeToFile(file, m); err != nil {
		t.Errorf("Failed to write to file: %v", err)
	}
}

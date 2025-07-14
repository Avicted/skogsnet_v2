package main

import (
	"fmt"
	"os"
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

func TestInsertMeasurement(t *testing.T) {
	tmpDB := "test_measurements.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	m := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 20.0,
		HumidityPercentage: 50.0,
	}
	if err := insertMeasurement(db, m, m.UnixTimestamp); err != nil {
		t.Errorf("Failed to insert measurement: %v", err)
	}

	// Verify that the measurement was inserted
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM measurements WHERE timestamp = ?", m.UnixTimestamp)
	if err := row.Scan(&count); err != nil {
		t.Errorf("Failed to query measurement: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 measurement in DB, got %d", count)
	}
}

func TestInsertMeasurement_InvalidDB(t *testing.T) {
	// Pass a nil db to insertMeasurement
	err := insertMeasurement(nil, Measurement{}, time.Now().UnixMilli())
	if err == nil {
		t.Error("Expected error when inserting with nil DB, got nil")
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

func TestEndToEndMeasurementFlow(t *testing.T) {
	tmpDB := "test_measurements_e2e.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	jsonStr := `{"temperature_celcius":21.1,"humidity":44.2}`
	m, err := deserializeData(jsonStr)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if err := insertMeasurement(db, m, m.UnixTimestamp); err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
	}

	var temp, hum float64
	row := db.QueryRow("SELECT temperature, humidity FROM measurements WHERE timestamp = ?", m.UnixTimestamp)
	if err := row.Scan(&temp, &hum); err != nil {
		t.Fatalf("Failed to query inserted measurement: %v", err)
	}
	if temp != 21.1 || hum != 44.2 {
		t.Errorf("Expected (21.1, 44.2), got (%v, %v)", temp, hum)
	}
}

func TestExportToCSV(t *testing.T) {
	tmpDB := "test_export.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	timestamp := time.Now().UnixMilli()

	// Insert some test data
	m1 := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 22.5,
		HumidityPercentage: 55.1,
	}

	weather := Weather{
		Name: "Vaasa",
		Main: struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		}{
			Temp:     24.5,
			Humidity: int(80.0),
		},
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		}{
			Speed: 5.0,
			Deg:   180,
		},
		Clouds: struct {
			All int `json:"all"`
		}{
			All: 75,
		},
		Weather: []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		}{
			{
				ID:          800,
				Main:        "Clear",
				Description: "clear sky",
			},
		},
	}

	if err := insertMeasurement(db, m1, timestamp); err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
	}

	if err := insertWeather(db, weather, timestamp); err != nil {
		t.Fatalf("Failed to insert weather: %v", err)
	}

	csvFile := "test_export.csv"
	if err := exportToCSV(db, csvFile); err != nil {
		t.Fatalf("Failed to export to CSV: %v", err)
	}
	defer os.Remove(csvFile)

	data, err := os.ReadFile(csvFile)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	expectedHeader := "timestamp,temperature,humidity,city,weather_temp,weather_humidity,wind_speed,wind_deg,clouds,weather_code,weather_description\n"
	if string(data[:len(expectedHeader)]) != expectedHeader {
		t.Errorf("CSV header mismatch:\nExpected: %q\nGot: %q", expectedHeader, string(data[:len(expectedHeader)]))
	}

	lines := string(data[len(expectedHeader):])
	lines = strings.TrimSuffix(lines, "\n") // Remove trailing newline for comparison

	if lines == "" {
		t.Error("CSV file should contain data after header, but it's empty")
	}

	// Check if the data matches the inserted measurement and weather
	expectedLine := fmt.Sprintf("%d,%.1f,%.1f,%s,%.1f,%d,%.1f,%d,%d,%d,%s",
		timestamp,
		m1.TemperatureCelsius,
		m1.HumidityPercentage,
		weather.Name,
		weather.Main.Temp,
		weather.Main.Humidity,
		weather.Wind.Speed,
		weather.Wind.Deg,
		weather.Clouds.All,
		weather.Weather[0].ID,
		weather.Weather[0].Description,
	)

	if lines != expectedLine {
		t.Errorf("CSV data mismatch:\nExpected: %q\nGot: %q", expectedLine, lines)
	}
}

func TestExportToCSV_EmptyDB(t *testing.T) {
	tmpDB := "test_empty_export.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	csvFile := "test_empty_export.csv"
	if err := exportToCSV(db, csvFile); err != nil {
		t.Fatalf("Failed to export to CSV: %v", err)
	}
	defer os.Remove(csvFile)

	data, err := os.ReadFile(csvFile)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	expectedHeader := "timestamp,temperature,humidity,city,weather_temp,weather_humidity,wind_speed,wind_deg,clouds,weather_code,weather_description\n"
	if string(data) != expectedHeader {
		t.Errorf("CSV file should only contain header, got: %q", string(data))
	}
}

func TestExportToCSV_NoWeatherData(t *testing.T) {
	tmpDB := "test_no_weather_export.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	m := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 22.0,
		HumidityPercentage: 60.0,
	}

	if err := insertMeasurement(db, m, m.UnixTimestamp); err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
	}

	csvFile := "test_no_weather_export.csv"
	if err := exportToCSV(db, csvFile); err != nil {
		t.Fatalf("Failed to export to CSV: %v", err)
	}
	defer os.Remove(csvFile)

	data, err := os.ReadFile(csvFile)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	expectedHeader := "timestamp,temperature,humidity,city,weather_temp,weather_humidity,wind_speed,wind_deg,clouds,weather_code,weather_description\n"
	if string(data[:len(expectedHeader)]) != expectedHeader {
		t.Errorf("CSV header mismatch:\nExpected: %q\nGot: %q", expectedHeader, string(data[:len(expectedHeader)]))
	}

	lines := string(data[len(expectedHeader):])
	lines = strings.TrimSuffix(lines, "\n") // Remove trailing newline for comparison

	if lines == "" {
		t.Error("CSV file should contain data after header, but it's empty")
	}

	expectedLine := fmt.Sprintf("%d,%.1f,%.1f,,0.0,0,0.0,0,0,0,", m.UnixTimestamp, m.TemperatureCelsius, m.HumidityPercentage)
	if lines != expectedLine {
		t.Errorf("CSV data mismatch:\nExpected: %q\nGot: %q", expectedLine, lines)
	}
}

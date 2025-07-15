package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

var origMustInitDatabase = mustInitDatabase
var origOpenDatabase = openDatabase

func mockMustInitDatabase(dbFileName *string) (*sql.DB, error) {
	return sql.Open("sqlite3", ":memory:")
}

func mockOpenDatabase(path string) (*sql.DB, error) {
	return sql.Open("sqlite3", ":memory:")
}

func TestMustInitDatabaseImpl_ValidPath(t *testing.T) {
	tmpDB := "test_real_mustinit.db"
	defer os.Remove(tmpDB)

	db, err := mustInitDatabaseImpl(&tmpDB)
	if err != nil {
		t.Fatalf("mustInitDatabaseImpl failed: %v", err)
	}
	if db == nil {
		t.Fatal("Expected non-nil DB from mustInitDatabaseImpl")
	}
	db.Close()
}

func TestMustInitDatabaseImpl_InvalidPath(t *testing.T) {
	invalidPath := "/invalid/path/to/db.db"
	db, err := mustInitDatabaseImpl(&invalidPath)
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
	if db != nil {
		t.Error("Expected nil DB for invalid path, got non-nil")
	}
}

func TestOpenDatabase_InvalidPath(t *testing.T) {
	invalidPath := "invalid/test.db"
	db, err := openDatabase(invalidPath)
	if err == nil {
		t.Errorf("Expected error when opening database with invalid path, got nil")
	}
	if db != nil {
		t.Errorf("Expected nil DB on error, got non-nil")
	}
}

func TestOpenDatabaseDBOpenFail(t *testing.T) {
	invalidPath := "/this/path/does/not/exist/test.db"

	db, err := openDatabase(invalidPath)
	if err == nil {
		t.Errorf("Expected error when opening non-existent database, got nil")
	}
	if db != nil {
		t.Errorf("Expected nil DB on error, got non-nil")
	}
}

func TestOpenDatabase_EmptyPath(t *testing.T) {
	db, err := openDatabase("")
	if err == nil {
		t.Errorf("Expected error for empty path, got nil")
	}
	if db != nil {
		t.Errorf("Expected nil DB for empty path, got non-nil")
	}
}

func TestEnableWALMode(t *testing.T) {
	tmpDB := "test_wal.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	enableWALMode(db)

	var mode string
	row := db.QueryRow("PRAGMA journal_mode")
	if err := row.Scan(&mode); err != nil {
		t.Errorf("Failed to query journal mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("Expected WAL mode, got %s", mode)
	}
}

func TestEnableWALMode_NilDB(t *testing.T) {
	exitCalled := false
	mockExit := func(code int) { exitCalled = true }
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	enableWALMode(nil)

	if !exitCalled {
		t.Error("Expected osExit to be called for nil DB")
	}
	if !strings.Contains(buf.String(), "Database connection is nil") {
		t.Error("Expected log error for nil DB")
	}
}

func TestEnableWALMode_ExecError(t *testing.T) {
	// Test: db.Exec error triggers osExit
	exitCalled := false
	mockExit := func(code int) { exitCalled = true }
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	// Use a closed DB to force Exec error
	tmpDB := "test_exec_error.db"
	defer os.Remove(tmpDB)
	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.Close() // Closed DB will cause Exec to fail

	enableWALMode(db)

	if !exitCalled {
		t.Error("Expected osExit to be called for Exec error")
	}
	if !strings.Contains(buf.String(), "Failed to enable WAL mode") {
		t.Error("Expected log error for Exec error")
	}
}

func TestEnableWALMode_VerificationError(t *testing.T) {
	// Test: WAL verification fails triggers osExit
	exitCalled := false
	mockExit := func(code int) { exitCalled = true }
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Set journal mode to something other than WAL
	if _, err := db.Exec("PRAGMA journal_mode = DELETE"); err != nil {
		t.Fatalf("Failed to set journal mode: %v", err)
	}

	enableWALMode(db)

	if !exitCalled {
		t.Error("Expected osExit to be called for WAL verification error")
	}
	if !strings.Contains(buf.String(), "WAL mode verification failed") {
		t.Error("Expected log error for WAL verification error")
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

func TestInsertMeasurement_WeatherQueryError(t *testing.T) {
	tmpDB := "test_weather_query_error.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Drop the weather table to cause a query error
	if _, err := db.Exec("DROP TABLE weather"); err != nil {
		t.Fatalf("Failed to drop weather table: %v", err)
	}

	m := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 21.0,
		HumidityPercentage: 45.0,
	}

	// This should fail because the weather table doesn't exist
	err = insertMeasurement(db, m, m.UnixTimestamp)
	if err == nil {
		t.Error("Expected error when querying non-existent weather table, got nil")
	}
	if err == sql.ErrNoRows {
		t.Error("Expected a real SQL error, got sql.ErrNoRows")
	}
}
func TestInsertWeather(t *testing.T) {
	tmpDB := "test_weather.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	weather := Weather{
		Name: "Test City",
		Main: struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		}{
			Temp:     22.5,
			Humidity: 60,
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
	}

	if err := insertWeather(db, weather, time.Now().UnixMilli()); err != nil {
		t.Errorf("Failed to insert weather: %v", err)
	}

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM weather")
	if err := row.Scan(&count); err != nil {
		t.Errorf("Failed to query weather count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 weather entry in DB, got %d", count)
	}
}

func TestInsertWeather_InvalidDB(t *testing.T) {
	// Pass a nil db to insertWeather
	err := insertWeather(nil, Weather{}, time.Now().UnixMilli())
	if err == nil {
		t.Error("Expected error when inserting with nil DB, got nil")
	}
}

func TestExportCSVAndExit(t *testing.T) {
	tmpDB := "test_export_csv.db"
	defer os.Remove(tmpDB)

	db, err := openDatabase(tmpDB)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Seed the database with some data
	timestamp := time.Now().UnixMilli()
	m1 := Measurement{
		UnixTimestamp:      timestamp,
		TemperatureCelsius: 22.5,
		HumidityPercentage: 55.1,
	}
	weather := Weather{
		Name: "Helsinki",
		Main: struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		}{
			Temp:     24.5,
			Humidity: 80,
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
	if err := insertWeather(db, weather, timestamp); err != nil {
		t.Fatalf("Failed to insert weather: %v", err)
	}
	if err := insertMeasurement(db, m1, timestamp); err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
	}

	csvFile := "test_export.csv"
	exportCSVAndExit(&tmpDB, &csvFile)

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
}

func TestExportCSVAndExit_OpenDatabaseError(t *testing.T) {
	// Simulate openDatabase error by passing an invalid path
	dbPath := "/invalid/path/to/db.db"
	csvPath := "should_not_be_created.csv"
	defer os.Remove(csvPath)

	exitCalled := false
	mockExit := func(code int) { exitCalled = true }
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	exportCSVAndExit(&dbPath, &csvPath)

	if !exitCalled {
		t.Error("Expected osExit to be called for openDatabase error")
	}
	if !strings.Contains(buf.String(), "Failed to open database") {
		t.Error("Expected log error for openDatabase error")
	}
}

func TestExportCSVAndExit_ExportError(t *testing.T) {
	openDatabase = mockOpenDatabase
	defer func() { openDatabase = origOpenDatabase }()

	// Simulate exportToCSV error by passing a nil DB pointer
	dbPath := "test_export_error.db"
	csvPath := "/invalid/path/to/export.csv" // Invalid path to trigger error
	defer os.Remove(dbPath)

	// Create a valid DB so openDatabase succeeds
	db, err := openDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.Close()

	exitCalled := false
	mockExit := func(code int) { exitCalled = true }
	originalExit := osExit
	osExit = mockExit
	defer func() { osExit = originalExit }()

	var buf bytes.Buffer
	log.SetOutput(&buf)

	exportCSVAndExit(&dbPath, &csvPath)

	if !exitCalled {
		t.Error("Expected osExit to be called for exportToCSV error")
	}
	if !strings.Contains(buf.String(), "Export to CSV failed") {
		t.Error("Expected log error for exportToCSV error")
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

	m1 := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 22.5,
		HumidityPercentage: 55.1,
	}

	weather := Weather{
		Name: "Helsinki",
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

	if err := insertWeather(db, weather, timestamp); err != nil {
		t.Fatalf("Failed to insert weather: %v", err)
	}

	if err := insertMeasurement(db, m1, timestamp); err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
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

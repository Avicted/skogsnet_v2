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
	if err := insertMeasurement(db, m); err != nil {
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
	err := insertMeasurement(nil, Measurement{})
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

	if err := insertMeasurement(db, m); err != nil {
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

	m1 := Measurement{
		UnixTimestamp:      time.Now().UnixMilli(),
		TemperatureCelsius: 20.0,
		HumidityPercentage: 50.0,
	}
	m2 := Measurement{
		UnixTimestamp:      time.Now().UnixMilli() + 1000,
		TemperatureCelsius: 22.0,
		HumidityPercentage: 55.0,
	}

	if err := insertMeasurement(db, m1); err != nil {
		t.Errorf("Failed to insert first measurement: %v", err)
	}
	if err := insertMeasurement(db, m2); err != nil {
		t.Errorf("Failed to insert second measurement: %v", err)
	}

	csvFile := "test_export.csv"
	if err := exportToCSV(db, csvFile); err != nil {
		t.Fatalf("Failed to export to CSV: %v", err)
	}
	defer os.Remove(csvFile)

	fileInfo, err := os.Stat(csvFile)
	if err != nil {
		t.Fatalf("CSV file does not exist after export: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Error("CSV file is empty after export")
	}
}

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

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	createTable := `
    CREATE TABLE IF NOT EXISTS measurements (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp INTEGER,
        temperature REAL,
        humidity REAL
    );`

	_, err = db.Exec(createTable)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func insertMeasurement(db *sql.DB, m Measurement) error {
	if db == nil {
		return errors.New("db is nil")
	}

	_, err := db.Exec(
		"INSERT INTO measurements (timestamp, temperature, humidity) VALUES (?, ?, ?)",
		m.UnixTimestamp, m.TemperatureCelsius, m.HumidityPercentage,
	)

	return err
}

func exportToCSV(db *sql.DB, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	rows, err := db.Query("SELECT timestamp, temperature, humidity FROM measurements ORDER BY timestamp")
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Fprintln(file, "timestamp,temperature,humidity")
	for rows.Next() {
		var ts int64
		var temp, hum float64
		if err := rows.Scan(&ts, &temp, &hum); err != nil {
			return err
		}
		fmt.Fprintf(file, "%d,%.2f,%.2f\n", ts, temp, hum)
	}

	return nil
}

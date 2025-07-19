package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStartDashboardServer(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Use a random port for testing
	port := 8080

	// Start the server in a goroutine
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startDashboardServer(ctx, db, &wg)
	}()
	wg.Wait()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/measurements", port))
	if err != nil {
		t.Fatalf("Failed to GET from dashboard server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestServeAPI_Endpoint(t *testing.T) {
	sqlDB, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer sqlDB.Close()

	gormDB, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}

	mux := http.NewServeMux()
	serveAPI(gormDB, mux)
	req := httptest.NewRequest("GET", "/api/measurements?range=1h", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestServeAPI_AllRanges(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}
	serveAPI(gormDB, mux)

	ranges := []string{"1h", "6h", "12h", "24h", "today", "week", "month", "year"}
	for _, r := range ranges {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/measurements?range=%s", r), nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected 200 OK for range '%s', got %d", r, w.Code)
		}
		if len(w.Body.Bytes()) == 0 {
			t.Errorf("Expected non-empty response body for range '%s'", r)
		}
	}
}

func TestServeAPI_InvalidRange(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}
	serveAPI(gormDB, mux)

	req := httptest.NewRequest("GET", "/api/measurements?range=invalid", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK for invalid range, returning all data, got %d", w.Code)
	}
}

func TestServeAPI_NoRange(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}
	serveAPI(gormDB, mux)
	req := httptest.NewRequest("GET", "/api/measurements", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK for no range, got %d", w.Code)
	}
	if len(w.Body.Bytes()) == 0 {
		t.Error("Expected non-empty response body for no range")
	}
}

func TestServeAPI_DBError(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Simulate a DB error by closing the connection
	db.Close()

	_, err = gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})

	if err == nil {
		t.Fatal("Failed to initialize GORM DB: sql: database is closed")
	}
}

func TestServeAPI_ScanRowsMapping(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS measurements (
			timestamp INTEGER,
			temperature REAL,
			humidity INTEGER,
			weather_id INTEGER
		);
		CREATE TABLE IF NOT EXISTS weather (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			city TEXT,
			temp REAL,
			humidity INTEGER,
			wind_speed REAL,
			wind_deg INTEGER,
			clouds INTEGER,
			weather_code INTEGER,
			description TEXT,
			timestamp INTEGER
		);
	`)
	if err != nil {
		t.Fatalf("Failed to setup tables: %v", err)
	}

	now := time.Now().UnixMilli()
	res, err := db.Exec(fmt.Sprintf(`
		INSERT INTO weather (city, temp, humidity, wind_speed, wind_deg, clouds, weather_code, description, timestamp)
			VALUES ('Helsinki', 22.1, 60, 5.5, 180, 75, 800, 'clear sky', %d);
	`, now))
	if err != nil {
		t.Fatalf("Failed to insert weather: %v", err)
	}
	weatherID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get weather id: %v", err)
	}
	_, err = db.Exec(fmt.Sprintf(`
		INSERT INTO measurements (timestamp, temperature, humidity, weather_id) VALUES (%d, 21.5, 55, %d);
	`, now, weatherID))
	if err != nil {
		t.Fatalf("Failed to insert measurement: %v", err)
	}

	mux := http.NewServeMux()
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}
	serveAPI(gormDB, mux)
	req := httptest.NewRequest("GET", "/api/measurements?range=1h", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	body := w.Body.Bytes()
	if len(body) == 0 {
		t.Fatal("Expected non-empty response body")
	}

	t.Logf("Response body: %s", string(body))

	// Check that the returned JSON contains the mapped fields
	if !strings.Contains(string(body), "Helsinki") ||
		!strings.Contains(string(body), "clear sky") ||
		!strings.Contains(string(body), "21.5") ||
		!strings.Contains(string(body), "22.1") {
		t.Errorf("Expected mapped fields in response, got: %s", string(body))
	}
}

func TestGetLatestMeasurement(t *testing.T) {
	db, err := openDatabase(":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Setup tables
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS measurements (
            timestamp INTEGER,
            temperature REAL,
            humidity INTEGER,
            weather_id INTEGER
        );
        CREATE TABLE IF NOT EXISTS weather (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            city TEXT,
            temp REAL,
            humidity INTEGER,
            wind_speed REAL,
            wind_deg INTEGER,
            clouds INTEGER,
            weather_code INTEGER,
            description TEXT,
            timestamp INTEGER
        );
    `)
	if err != nil {
		t.Fatalf("Failed to setup tables: %v", err)
	}

	// Insert a weather row
	now := time.Now().UnixMilli()
	res, err := db.Exec(`
        INSERT INTO weather (city, temp, humidity, wind_speed, wind_deg, clouds, weather_code, description, timestamp)
        VALUES ('Helsinki', 22.1, 60, 5.5, 180, 75, 800, 'clear sky', ?)`, now)
	if err != nil {
		t.Fatalf("Failed to insert weather: %v", err)
	}
	weatherID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get weather id: %v", err)
	}

	// Insert 10 measurements with increasing temperature
	for i := 0; i < 10; i++ {
		_, err = db.Exec(`
            INSERT INTO measurements (timestamp, temperature, humidity, weather_id)
            VALUES (?, ?, ?, ?)`,
			now-int64((9-i)*60000), // 1 minute apart, oldest first
			20.0+float64(i),        // temperature: 20.0, 21.0, ..., 29.0
			50+i,
			weatherID,
		)
		if err != nil {
			t.Fatalf("Failed to insert measurement %d: %v", i, err)
		}
	}

	mux := http.NewServeMux()
	gormDB, err := gorm.Open(sqlite.Dialector{Conn: db}, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to initialize GORM DB: %v", err)
	}
	serveAPI(gormDB, mux)

	req := httptest.NewRequest("GET", "/api/measurements/latest", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
	body := w.Body.Bytes()
	if len(body) == 0 {
		t.Error("Expected non-empty response body for latest measurement")
	}

	// Parse response and check trajectory
	var resp struct {
		Latest     map[string]any `json:"latest"`
		Trajectory float64        `json:"trajectory"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// The trajectory should be 29.0 - 20.0 = 9.0
	if resp.Trajectory != 9.0 {
		t.Errorf("Expected trajectory 9.0, got %v", resp.Trajectory)
	}
	if resp.Latest == nil {
		t.Error("Expected latest measurement in response")
	}
}

package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

var originalHTTPGet func(string) (*http.Response, error)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Helper for substring check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestStartWeatherFetcher_InsertsWeather(t *testing.T) {
	// Mock the HTTP-dependent functions
	GetCityLatLong = func(city string) (GeoResponse, error) {
		if city == "" || city == "InvalidCityName12345" {
			return GeoResponse{}, fmt.Errorf("no results found for city: %s", city)
		}
		return GeoResponse{
			Results: []GeoResult{{Name: city, Latitude: 60.0, Longitude: 25.0}},
		}, nil
	}
	GetWeatherData = func(city string) (Weather, error) {
		return Weather{
			Name: city,
			Main: struct {
				Temp     float64 `json:"temp"`
				Humidity int     `json:"humidity"`
			}{Temp: 20.0, Humidity: 50},
			Wind: struct {
				Speed float64 `json:"speed"`
				Deg   int     `json:"deg"`
			}{Speed: 5.0, Deg: 90},
			Clouds: struct {
				All int `json:"all"`
			}{All: 10},
			Weather: []struct {
				ID          int    `json:"id"`
				Main        string `json:"main"`
				Description string `json:"description"`
			}{{ID: 800, Main: "Clear", Description: "clear sky"}},
		}, nil
	}

	// Use an in-memory SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	// Create weather table (minimal schema for insertWeather)
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS weather (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            timestamp INTEGER,
            city TEXT,
            temp REAL,
            humidity INTEGER,
            wind_speed REAL,
            wind_deg INTEGER,
            clouds INTEGER,
            weather_code INTEGER,
            description TEXT
        );
    `)
	if err != nil {
		t.Fatalf("Failed to create weather table: %v", err)
	}

	// Set required global flag
	city := "Helsinki"
	weatherCity = &city

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the fetcher
	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)

	// Wait for initial fetch and insert
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()

	// Check that weather was inserted
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM weather WHERE city = ?", city)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to query weather table: %v", err)
	}
	if count == 0 {
		t.Error("Expected at least one weather row inserted")
	}
}

func TestStartWeatherFetcher_EmptyCity(t *testing.T) {
	// Save and restore original osExit
	origOsExit := osExit
	defer func() { osExit = origOsExit }()

	// Mock osExit to panic so we can catch it
	osExit = func(code int) { panic(fmt.Sprintf("osExit called with code %d", code)) }

	// Set weatherCity to empty
	weatherCity = new(string)
	*weatherCity = ""

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected osExit to be called, but it was not")
		}
	}()

	startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
}

func TestStartWeatherFetcher_ContextCancel(t *testing.T) {

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	weatherCity = new(string)
	*weatherCity = "Helsinki"

	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)

	// Cancel context to trigger exit
	cancel()
	wg.Wait()

	// Assert that the fetcher exits cleanly
	select {
	case <-time.After(1 * time.Second):
		t.Error("Expected fetcher to exit cleanly after context cancel, but it did not")
	default:
		// Fetcher exited as expected
	}
}

func TestStartWeatherFetcher_InsertWeatherError(t *testing.T) {
	// Save and restore original logError
	origLogError := logError
	called := false
	var loggedMsg string
	logError = func(format string, args ...interface{}) {
		called = true
		loggedMsg = fmt.Sprintf(format, args...)
	}
	defer func() { logError = origLogError }()

	// Set up a DB that will fail on insert
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	// Create a weather table with missing columns to force an insert error
	_, err = db.Exec(`CREATE TABLE weather (id INTEGER PRIMARY KEY)`)
	if err != nil {
		t.Fatalf("Failed to create broken weather table: %v", err)
	}

	city := "Helsinki"
	weatherCity = &city

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the fetcher (should fail to insert)
	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	time.Sleep(1 * time.Second)
	cancel()
	wg.Wait()

	// Assert that the error was logged
	if !called {
		t.Error("Expected logError to be called on insert error")
	}
	if loggedMsg == "" || !contains(loggedMsg, "Failed to insert initial weather data") {
		t.Errorf("Expected logError message about insert failure, got: %s", loggedMsg)
	}
}

func TestStartWeatherFetcher_InitialFetchRetry(t *testing.T) {
	// Save and restore original logError and GetWeatherData
	origLogError := logError
	origGetWeatherData := GetWeatherData
	called := false
	var loggedMsg string
	logError = func(format string, args ...interface{}) {
		called = true
		loggedMsg = fmt.Sprintf(format, args...)
	}
	GetWeatherData = func(city string) (Weather, error) {
		return Weather{}, fmt.Errorf("simulated fetch error")
	}
	defer func() {
		logError = origLogError
		GetWeatherData = origGetWeatherData
	}()

	// Use an in-memory SQLite DB
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	// Create weather table (minimal schema for insertWeather)
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS weather (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            timestamp INTEGER,
            city TEXT,
            temp REAL,
            humidity INTEGER,
            wind_speed REAL,
            wind_deg INTEGER,
            clouds INTEGER,
            weather_code INTEGER,
            description TEXT
        );
    `)
	if err != nil {
		t.Fatalf("Failed to create weather table: %v", err)
	}

	city := "Helsinki"
	weatherCity = &city

	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the fetcher (should fail and retry)
	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	time.Sleep(1 * time.Second) // Give it time to hit the error and retry
	cancel()
	wg.Wait()

	// Assert that the error was logged
	if !called {
		t.Error("Expected logError to be called on initial fetch error")
	}
	if loggedMsg == "" || !contains(loggedMsg, "Initial weather fetch failed, retrying in 5s") {
		t.Errorf("Expected logError message about initial fetch retry, got: %s", loggedMsg)
	}
}

func TestStartWeatherFetcher_Ticker_Success(t *testing.T) {
	origGetWeatherData := GetWeatherData
	origInsertWeather := insertWeather

	GetWeatherData = func(city string) (Weather, error) {
		return Weather{Name: city}, nil
	}
	insertWeather = func(db *sql.DB, w Weather, ts int64) error {
		return nil
	}

	defer func() {
		GetWeatherData = origGetWeatherData
		insertWeather = origInsertWeather
	}()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	city := "Helsinki"
	weatherCity = &city
	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	weatherTickerInterval = 500 * time.Millisecond // Shorten ticker interval for test speed

	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()

	if latestWeather.Name != city {
		t.Errorf("Expected latest weather city to be %s, got %s", city, latestWeather.Name)
	}
}

func TestStartWeatherFetcher_Ticker_InsertError(t *testing.T) {
	origGetWeatherData := GetWeatherData
	origInsertWeather := insertWeather
	origThrottledLogError := throttledLogError

	GetWeatherData = func(city string) (Weather, error) {
		return Weather{Name: city}, nil
	}
	insertWeatherCalled := false
	insertWeather = func(db *sql.DB, w Weather, ts int64) error {
		insertWeatherCalled = true
		return fmt.Errorf("insert error")
	}
	throttledLogErrorCalled := false
	throttledLogError = func(last *time.Time, format string, args ...interface{}) {
		throttledLogErrorCalled = true
		if !strings.Contains(format, "Failed to insert weather data") {
			t.Errorf("Expected insert error log, got: %s", format)
		}
	}
	defer func() {
		GetWeatherData = origGetWeatherData
		insertWeather = origInsertWeather
		throttledLogError = origThrottledLogError
	}()

	db, _ := sql.Open("sqlite3", ":memory:")
	city := "Helsinki"
	weatherCity = &city
	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	weatherTickerInterval = 500 * time.Millisecond
	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	time.Sleep(1 * time.Second)
	cancel()
	wg.Wait()

	if !insertWeatherCalled {
		t.Error("Expected insertWeather to be called")
	}
	if !throttledLogErrorCalled {
		t.Error("Expected throttledLogError to be called for insert error")
	}
}

func TestStartWeatherFetcher_Ticker_GetWeatherError(t *testing.T) {
	origGetWeatherData := GetWeatherData
	origThrottledLogError := throttledLogError

	callCount := 0
	GetWeatherData = func(city string) (Weather, error) {
		callCount++
		if callCount == 1 {
			return Weather{Name: city}, nil // Initial fetch succeeds
		}
		return Weather{}, fmt.Errorf("fetch error") // Ticker fetch fails
	}
	throttledLogErrorCalled := false
	throttledLogError = func(last *time.Time, format string, args ...interface{}) {
		throttledLogErrorCalled = true
		if !strings.Contains(format, "Failed to get weather data for city") {
			t.Errorf("Expected get weather error log, got: %s", format)
		}
	}
	defer func() {
		GetWeatherData = origGetWeatherData
		throttledLogError = origThrottledLogError
	}()

	db, _ := sql.Open("sqlite3", ":memory:")
	city := "Helsinki"
	weatherCity = &city
	var latestWeather Weather
	var latestWeatherTimestamp int64
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	weatherTickerInterval = 500 * time.Millisecond
	go startWeatherFetcher(ctx, db, &latestWeather, &latestWeatherTimestamp, &wg)
	time.Sleep(1 * time.Second)
	cancel()
	wg.Wait()

	if !throttledLogErrorCalled {
		t.Error("Expected throttledLogError to be called for GetWeatherData error")
	}
}

func TestWindDirectionToCompass(t *testing.T) {
	tests := []struct {
		degrees int
		compass string
	}{
		{0, "N"},
		{45, "NE"},
		{90, "E"},
		{135, "SE"},
		{180, "S"},
		{225, "SW"},
		{270, "W"},
		{315, "NW"},
	}

	for _, test := range tests {
		result := WindDirectionToCompass(test.degrees)
		if result != test.compass {
			t.Errorf("Expected %s for %d degrees, got %s", test.compass, test.degrees, result)
		}
	}
}

func TestWindDirectionToCompassInvalid(t *testing.T) {
	tests := []int{-1, 360, 400, -100}

	for _, deg := range tests {
		result := WindDirectionToCompass(deg)
		if result != "" {
			t.Errorf("Expected empty string for %d degrees, got %s", deg, result)
		}
	}
}

func TestWeatherCodeToSentence_AllCases(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{0, "Clear sky"},
		{1, "Mainly clear"},
		{2, "Partly cloudy"},
		{3, "Overcast"},
		{45, "Fog"},
		{48, "Depositing rime fog"},
		{51, "Light drizzle"},
		{53, "Moderate drizzle"},
		{55, "Dense drizzle"},
		{56, "Light freezing drizzle"},
		{57, "Dense freezing drizzle"},
		{61, "Slight rain"},
		{63, "Moderate rain"},
		{65, "Heavy rain"},
		{66, "Light freezing rain"},
		{67, "Heavy freezing rain"},
		{71, "Slight snow fall"},
		{73, "Moderate snow fall"},
		{75, "Heavy snow fall"},
		{77, "Snow grains"},
		{80, "Slight rain showers"},
		{81, "Moderate rain showers"},
		{82, "Violent rain showers"},
		{85, "Slight snow showers"},
		{86, "Heavy snow showers"},
		{95, "Thunderstorm"},
		{96, "Thunderstorm with slight hail"},
		{99, "Thunderstorm with heavy hail"},
		{-1, "Unknown weather code"},
		{999, "Unknown weather code"},
	}

	for _, test := range tests {
		result := WeatherCodeToSentence(test.code)
		if result != test.expected {
			t.Errorf("For code %d, expected '%s', got '%s'", test.code, test.expected, result)
		}
	}
}

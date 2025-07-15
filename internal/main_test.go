package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
)

// Mock serial.Port for mainLoop
type mockSerialPort struct {
	data []string
	idx  int
}

// GetModemStatusBits implements serial.Port.
func (m *mockSerialPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	panic("unimplemented")
}

// ResetInputBuffer implements serial.Port.
func (m *mockSerialPort) ResetInputBuffer() error {
	panic("unimplemented")
}

// ResetOutputBuffer implements serial.Port.
func (m *mockSerialPort) ResetOutputBuffer() error {
	panic("unimplemented")
}

// SetDTR implements serial.Port.
func (m *mockSerialPort) SetDTR(dtr bool) error {
	panic("unimplemented")
}

// SetMode implements serial.Port.
func (m *mockSerialPort) SetMode(mode *serial.Mode) error {
	panic("unimplemented")
}

// SetRTS implements serial.Port.
func (m *mockSerialPort) SetRTS(rts bool) error {
	panic("unimplemented")
}

// SetReadTimeout implements serial.Port.
func (m *mockSerialPort) SetReadTimeout(t time.Duration) error {
	panic("unimplemented")
}

func (m *mockSerialPort) Read(p []byte) (int, error) {
	if m.idx >= len(m.data) {
		return 0, nil
	}
	n := copy(p, m.data[m.idx]+"\n")
	m.idx++
	return n, nil
}
func (m *mockSerialPort) Write(p []byte) (int, error)        { return len(p), nil }
func (m *mockSerialPort) Close() error                       { return nil }
func (m *mockSerialPort) Break(duration time.Duration) error { return nil }
func (m *mockSerialPort) Drain() error                       { return nil }

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

func TestMainLoop_GracefulShutdown(t *testing.T) {
	serialPort := &mockSerialPort{data: []string{"{\"temperature_celcius\":21.1,\"humidity\":44.2}"}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should exit gracefully, no panic
}

func TestMainLoop_TimeoutError(t *testing.T) {
	origReadFromSerial := readFromSerial
	readFromSerial = func(scanner Scanner) (string, error) {
		return "", errors.New("timeout")
	}
	defer func() { readFromSerial = origReadFromSerial }()

	serialPort := &mockSerialPort{data: []string{}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should handle timeout error and continue
}

func TestMainLoop_EmptyLine(t *testing.T) {
	origReadFromSerial := readFromSerial
	readFromSerial = func(scanner Scanner) (string, error) {
		return "", nil
	}
	defer func() { readFromSerial = origReadFromSerial }()

	serialPort := &mockSerialPort{data: []string{}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should handle empty line and continue
}

func TestMainLoop_DeserializeError(t *testing.T) {
	origReadFromSerial := readFromSerial
	origDeserializeData := deserializeData
	readFromSerial = func(scanner Scanner) (string, error) {
		return "bad json", nil
	}
	deserializeData = func(s string) (Measurement, error) {
		return Measurement{}, errors.New("deserialize error")
	}
	defer func() {
		readFromSerial = origReadFromSerial
		deserializeData = origDeserializeData
	}()

	serialPort := &mockSerialPort{data: []string{"bad json"}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should handle deserialize error and continue
}

func TestMainLoop_InsertError(t *testing.T) {
	origReadFromSerial := readFromSerial
	origDeserializeData := deserializeData
	origInsertMeasurement := insertMeasurement
	readFromSerial = func(scanner Scanner) (string, error) {
		return "{\"temperature_celcius\":21.1,\"humidity\":44.2}", nil
	}
	deserializeData = func(s string) (Measurement, error) {
		return Measurement{TemperatureCelsius: 21.1, HumidityPercentage: 44.2}, nil
	}
	insertMeasurement = func(db *sql.DB, m Measurement, ts int64) error {
		return errors.New("insert error")
	}
	defer func() {
		readFromSerial = origReadFromSerial
		deserializeData = origDeserializeData
		insertMeasurement = origInsertMeasurement
	}()

	serialPort := &mockSerialPort{data: []string{"{\"temperature_celcius\":21.1,\"humidity\":44.2}"}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should handle insert error and continue
}

func TestMainLoop_Success(t *testing.T) {
	origReadFromSerial := readFromSerial
	origDeserializeData := deserializeData
	origInsertMeasurement := insertMeasurement
	origPrintToConsole := printToConsole
	readFromSerial = func(scanner Scanner) (string, error) {
		return "{\"temperature_celcius\":21.1,\"humidity\":44.2}", nil
	}
	deserializeData = func(s string) (Measurement, error) {
		return Measurement{TemperatureCelsius: 21.1, HumidityPercentage: 44.2}, nil
	}
	insertMeasurement = func(db *sql.DB, m Measurement, ts int64) error {
		return nil
	}
	printToConsole = func(m Measurement, w *Weather) {}
	defer func() {
		readFromSerial = origReadFromSerial
		deserializeData = origDeserializeData
		insertMeasurement = origInsertMeasurement
		printToConsole = origPrintToConsole
	}()

	serialPort := &mockSerialPort{data: []string{"{\"temperature_celcius\":21.1,\"humidity\":44.2}"}}
	db, _ := sql.Open("sqlite3", ":memory:")
	var latestWeather Weather
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	mainLoop(ctx, serialPort, db, &latestWeather, &wg)
	// Should process successfully
}

/* func TestMain_ExportCSV(t *testing.T) {
	origExportCSVAndExit := exportCSVAndExit
	origMustInitDatabase := mustInitDatabase
	origExportCSV := *exportCSV

	exportCSVAndExitCalled := false
	exportCSVAndExit = func(dbFileName, exportCSV *string) {
		exportCSVAndExitCalled = true
	}
	mustInitDatabase = func(dbFileName *string) (*sql.DB, error) {
		return &sql.DB{}, nil // Return a dummy DB
	}
	*exportCSV = "out.csv"
	defer func() {
		exportCSVAndExit = origExportCSVAndExit
		mustInitDatabase = origMustInitDatabase
		*exportCSV = origExportCSV // Reset to original value
	}()

	main()
	if !exportCSVAndExitCalled {
		t.Error("Expected exportCSVAndExit to be called")
	}
} */

func TestMain_ExportCSV(t *testing.T) {
	origExportCSVAndExit := exportCSVAndExit
	origSetupLogging := setupLogging
	origExportCSV := *exportCSV

	exportCSVAndExitCalled := false
	exportCSVAndExit = func(dbFileName, exportCSV *string) {
		exportCSVAndExitCalled = true
	}
	setupLogging = func() {}
	*exportCSV = "out.csv"
	defer func() {
		exportCSVAndExit = origExportCSVAndExit
		setupLogging = origSetupLogging
		*exportCSV = origExportCSV
	}()

	main()
	if !exportCSVAndExitCalled {
		t.Error("Expected exportCSVAndExit to be called")
	}
}

func TestMain_DBInitError(t *testing.T) {
	origMustInitDatabase := mustInitDatabase
	origSetupLogging := setupLogging
	origLogFatal := logFatal
	origOsExit := osExit
	origInitSerialPort := initSerialPort

	mustInitDatabase = func(dbFileName *string) (*sql.DB, error) {
		return nil, errors.New("db error")
	}
	setupLogging = func() {}
	logFatalCalled := false
	logFatal = func(format string, args ...interface{}) { logFatalCalled = true }
	osExitCalled := false
	osExit = func(code int) { osExitCalled = true }
	initSerialPort = func() serial.Port { return &mockSerialPort{} }

	defer func() {
		mustInitDatabase = origMustInitDatabase
		setupLogging = origSetupLogging
		logFatal = origLogFatal
		osExit = origOsExit
		initSerialPort = origInitSerialPort
	}()

	*exportCSV = ""
	main()
	if !logFatalCalled || !osExitCalled {
		t.Error("Expected logFatal and osExit to be called on DB error")
	}
}

func TestMain_WeatherFetcherEnabled(t *testing.T) {
	origStartWeatherFetcher := startWeatherFetcher
	origSetupLogging := setupLogging
	origMustInitDatabase := mustInitDatabase
	origInitSerialPort := initSerialPort
	origEnableWALMode := enableWALMode
	origMainLoop := mainLoop
	origExportCSV := *exportCSV
	origEnableWeather := *enableWeather

	startWeatherFetcherCalled := false
	startWeatherFetcher = func(ctx context.Context, db *sql.DB, latestWeather *Weather, latestWeatherTimestamp *int64, wg *sync.WaitGroup) {
		startWeatherFetcherCalled = true
	}
	setupLogging = func() {}
	mustInitDatabase = func(dbFileName *string) (*sql.DB, error) {
		return sql.Open("sqlite3", ":memory:")
	}
	initSerialPort = func() serial.Port { return &mockSerialPort{} }
	enableWALMode = func(db *sql.DB) {}
	mainLoop = func(ctx context.Context, serialPort serial.Port, db *sql.DB, latestWeather *Weather, wg *sync.WaitGroup) {
	}
	*exportCSV = ""
	*enableWeather = true
	defer func() {
		startWeatherFetcher = origStartWeatherFetcher
		setupLogging = origSetupLogging
		mustInitDatabase = origMustInitDatabase
		initSerialPort = origInitSerialPort
		enableWALMode = origEnableWALMode
		mainLoop = origMainLoop
		*exportCSV = origExportCSV
		*enableWeather = origEnableWeather
	}()

	main()
	if !startWeatherFetcherCalled {
		t.Error("Expected startWeatherFetcher to be called when enabled")
	}
}

func TestMain_DashboardEnabled(t *testing.T) {
	origStartDashboardServer := startDashboardServer
	origSetupLogging := setupLogging
	origMustInitDatabase := mustInitDatabase
	origInitSerialPort := initSerialPort
	origEnableWALMode := enableWALMode
	origMainLoop := mainLoop
	origExportCSV := *exportCSV
	origServeDashboard := *serveDashboard

	startDashboardServerCalled := false
	startDashboardServer = func(ctx context.Context, db *sql.DB, wg *sync.WaitGroup) {
		startDashboardServerCalled = true
	}
	setupLogging = func() {}
	mustInitDatabase = func(dbFileName *string) (*sql.DB, error) {
		return sql.Open("sqlite3", ":memory:")
	}
	initSerialPort = func() serial.Port { return &mockSerialPort{} }
	enableWALMode = func(db *sql.DB) {}
	mainLoop = func(ctx context.Context, serialPort serial.Port, db *sql.DB, latestWeather *Weather, wg *sync.WaitGroup) {
	}
	*exportCSV = ""
	*serveDashboard = true
	defer func() {
		startDashboardServer = origStartDashboardServer
		setupLogging = origSetupLogging
		mustInitDatabase = origMustInitDatabase
		initSerialPort = origInitSerialPort
		enableWALMode = origEnableWALMode
		*exportCSV = origExportCSV
		*serveDashboard = origServeDashboard
		mainLoop = origMainLoop
	}()

	main()
	if !startDashboardServerCalled {
		t.Error("Expected startDashboardServer to be called when enabled")
	}
}
func TestMain_MainLoopRuns(t *testing.T) {
	origMainLoop := mainLoop
	origSetupLogging := setupLogging
	origMustInitDatabase := mustInitDatabase
	origInitSerialPort := initSerialPort
	origEnableWALMode := enableWALMode
	origExportCSV := *exportCSV

	mainLoopCalled := false
	mainLoop = func(ctx context.Context, serialPort serial.Port, db *sql.DB, latestWeather *Weather, wg *sync.WaitGroup) {
		mainLoopCalled = true
	}
	setupLogging = func() {}
	mustInitDatabase = func(dbFileName *string) (*sql.DB, error) {
		return sql.Open("sqlite3", ":memory:")
	}
	initSerialPort = func() serial.Port { return &mockSerialPort{} }
	enableWALMode = func(db *sql.DB) {}
	*exportCSV = ""
	defer func() {
		mainLoop = origMainLoop
		setupLogging = origSetupLogging
		mustInitDatabase = origMustInitDatabase
		initSerialPort = origInitSerialPort
		enableWALMode = origEnableWALMode
		*exportCSV = origExportCSV
	}()

	main()
	if !mainLoopCalled {
		t.Error("Expected mainLoop to be called")
	}
}

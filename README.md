# Skogsnet v2

Skogsnet v2 is a Go application for reading temperature and humidity measurements from a serial device, saving them to a SQLite database, and printing them to the console in a human-readable format.

## Features

- Reads JSON-formatted measurements from a serial port (default: `/dev/ttyACM0`)
- Saves measurements to a SQLite database (`measurements.db`)
- Prints each measurement to the console with a readable timestamp
- Handles graceful shutdown on Ctrl+C or SIGTERM

## Requirements

- Go 1.18 or newer
- [go.bug.st/serial](https://github.com/bugst/go-serial)
- [github.com/mattn/go-sqlite3](https://pkg.go.dev/github.com/mattn/go-sqlite3)

## Build

```sh
mkdir -p build
go build -o build/skogsnet_v2 ./internal
```

## Test
```sh
go test ./internal
```

## Run

```sh
./build/skogsnet_v2
```

## Output

- Measurements are stored in a SQLite database file named `measurements.db`.
- Console output example:
  ```
  Measurement at 2025-07-13 14:23:15: Temperature = 23.66 °C, Humidity = 77.38%
  ```

## Configuration

- The serial port is currently hardcoded to `/dev/ttyACM0`.  
  To change, edit the `portName` constant in `internal/main.go`.
- The baud rate is set to 9600.
- The measurement database file is named `measurements.db`
  To change, edit the `measurementFileName` constant in `internal/main.go`.

## License
MIT License
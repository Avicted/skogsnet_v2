# Skogsnet v2

Skogsnet v2 is a Go application for reading temperature and humidity measurements from a serial device, saving them to a file, and printing them to the console in a human-readable format.

## Features

- Reads JSON-formatted measurements from a serial port (default: `/dev/ttyACM0`)
- Saves measurements to `measurements.dat` with a header and millisecond timestamps
- Prints each measurement to the console with a readable timestamp
- Handles graceful shutdown on Ctrl+C or SIGTERM

## Requirements

- Go 1.18 or newer
- [go.bug.st/serial](https://github.com/bugst/go-serial) library

## Build

```sh
mkdir -p build
go build -o build/skogsnet_v2 ./internal
```

## Run

```sh
./build/skogsnet_v2
```

## Output

- Measurements are appended to `measurements.dat` in the working directory.
- Console output example:
  ```
  Measurement at 2025-07-13 14:23:15: Temperature = 23.66 °C, Humidity = 77.38%
  ```

## Configuration

- The serial port is currently hardcoded to `/dev/ttyACM0`.  
  To change, edit the `portName` constant in `internal/main.go`.
- The baud rate is set to 9600.
- The measurement file is set to `measurements.dat`.  
  To change, edit the `measurementFileName` constant in `internal/main.go`.

## License
MIT License
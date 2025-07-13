# Skogsnet v2

Skogsnet v2 is a Go application for reading temperature and humidity measurements from a serial device, saving them to a SQLite database, and printing them to the console in a human-readable format.

## Features

- Reads JSON-formatted measurements from a serial port (default: `/dev/ttyACM0`)
- Saves measurements to a SQLite database (`measurements.db`)
- Prints each measurement to the console with a readable timestamp
- Handles graceful shutdown on Ctrl+C or SIGTERM
- Exports measurements to a CSV file if specified

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

## Configuration

You can set the serial port and baud rate using command-line flags:

```sh
./build/skogsnet_v2 --port=/dev/ttyUSB0 --baud=115200
```

- `--port` sets the serial port device (default: `/dev/ttyACM0`)
- `--baud` sets the baud rate (default: `9600`)

The database filename is set by the `dbFileName` variable in the code

## Run

```sh
./build/skogsnet_v2
```

## Output

- Measurements are stored in a SQLite database file named `measurements.db`.
- Console output example:
  ```
  Measurement at 2025-07-13 18:23:12
    Temperature: 23.78 °C
    Humidity:    74.44 %
  ```


## Export to CSV
You can export measurements to a CSV file by using the `--export-csv` flag:

```sh
./build/skogsnet_v2 --export-csv=measurements.csv
```


## License
MIT License
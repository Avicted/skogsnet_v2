# Skogsnet v2

Skogsnet v2 is a Go application for reading temperature and humidity measurements from a serial device, saving them to a SQLite database, and printing them to the console in a human-readable format.

## Features

- Reads JSON-formatted measurements from a serial port (default: `/dev/ttyACM0`)
- Saves measurements to a SQLite database (`measurements.db`)
- Prints each measurement to the console with a readable timestamp
- Handles graceful shutdown on Ctrl+C or SIGTERM
- Exports measurements to a CSV file if specified
- Logs to a file if specified

## Requirements

- Go 1.18 or newer


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

The following CLI flags are available:

```sh
Usage of ./build/skogsnet_v2:
  -baud int
    	Serial baud rate (default 9600)
  -db string
    	SQLite database filename (default "measurements.db")
  -export-csv string
    	Export measurements to CSV file and exit
  -log-file string
    	Log output to file (optional)
  -port string
    	Serial port name (default "/dev/ttyACM0")
```



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
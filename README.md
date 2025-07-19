# Skogsnet v2

Skogsnet v2 is a Go application for reading temperature and humidity measurements from a serial device, saving them to a SQLite database. It features a web dashboard for visualizing the data, configurable logging, and the ability to export measurements to CSV. It also integrates with the OpenMeteo API to fetch current weather data, displaying it alongside the measurements.


## Features

- **Serial Data Acquisition:** Reads JSON-formatted measurements from a serial port (default: `/dev/ttyACM0`)
- **Database Storage:** Saves measurements to a SQLite database (`measurements.db`)
- **Console Output:** Prints each measurement in a readable, color-formatted style
- **Graceful Shutdown:** Handles Ctrl+C or SIGTERM cleanly
- **CSV Export:** Export all measurements to a CSV file with a single flag
- **Configurable Logging:** Log to a file with log levels (info, warn, error)
- **Web Dashboard:** Visualize measurements with an interactive chart and time range selection with dark mode support
- **Weather Data Integration:** Fetches current weather data from OpenMeteo API and displays it alongside measurements
- **Docker Support:** Easily deploy with Docker and Docker Compose

## Requirements

- Go 1.24.5 or newer
- Serial device providing JSON-formatted temperature and humidity data

- Optional:
  - Node.js and npm (for building the web dashboard)


## Build

```sh
mkdir -p build
go build -o build/skogsnet_v2 ./internal
```

## Test
```sh
# Run all tests
go test ./internal

# Run tests with coverage
go test -coverprofile=coverage.out ./internal

# View coverage report
go tool cover -html=coverage.out
```

## Configuration

The following CLI flags are available:

```
./build/skogsnet_v2 -h

Usage of ./build/skogsnet_v2:
  -baud int
    	Serial baud rate (default 9600)
  -city string
    	City name for weather data
  -dashboard
    	Serve web dashboard at http://localhost:8080
  -db string
    	SQLite database filename (default "measurements.db")
  -export-csv string
    	Export measurements to CSV file and exit
  -log-file string
    	Log output to file (optional)
  -port string
    	Serial port name (default "/dev/ttyACM0")
  -weather
    	Enable periodic weather data fetching
```


## Run

```sh
# Only sensor data acquisition and storage
./build/skogsnet_v2

# With logging, weather data, and dashboard enabled
./build/skogsnet_v2 -log-file=skogsnet.log -dashboard -weather -city=Helsinki
```

## Output

- Measurements are stored in a SQLite database file named `measurements.db`.
- Console output example:
```
Measurement at 2025-07-14 16:50:08
    Temperature:         23.78 °C
    Humidity:            76.06 %

    Weather:             Overcast
    Outside Temperature: 27.00 °C
    Outside Humidity:    44%
    Wind Speed:          6.60 m/s
    Wind Direction:      22° N
    Cloud Cover:         0%
```

## Web Dashboard

- **Features:**  
  - Interactive chart
  - Time range selection: 1h, 6h, 12h, 24h, today, week, month, year, all
  - Data:
    - Temperature (Sensor and outside)
    - Humidity (Sensor and outside)
    - Wind speed
    - Weather description
    - Delta temperature (trajectory)
  - Live data updates (toggleable)
  - Responsive design
  - Dark mode support

- **Build**
  ```sh
  cd skogsnet-frontend
  npm install
  npm run build
  ```

- **Access:**
  Start with `-dashboard` and open [http://localhost:8080](http://localhost:8080)

![web-dashboard](skogsnet-frontend/react-frontend-screenshot.png)


## Docker Support
Please make your desired changes to the `docker-compose.yml` file to set the environment variables and serial port.


Build and Run with Docker Compose:
```sh
docker compose up --build
```

### Host Directory for Database and Log Files
If you want the database and log files to exist in a host directory, you can create a `data` directory in the same location as the `docker-compose.yml` file and mount it as a volume:

Create the `data` directory and files + set permissions:
```sh
mkdir -p ./data
touch ./data/measurements-docker.db
touch ./data/skogsnet-docker.log
chmod 666 ./data/measurements-docker.db ./data/skogsnet-docker.log
```

In `docker-compose.yml` add the volumes mount:
```yaml
services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile
    devices:
      - "/dev/ttyACM0:/dev/ttyACM0"            # Assign to your serial port
    environment:                               # Options for the program
      BAUD: "9600"
      CITY: "Helsinki"
      DASHBOARD: "true"
      DB: "measurements-docker.db"
      EXPORT_CSV: ""
      LOG_FILE: "skogsnet-docker.log"
      PORT: "/dev/ttyACM0"
      WEATHER: "true"
    ports:
      - "8080:8080"
    volumes:                                    # Store the database and log outside the container
      - ./data/measurements-docker.db:/app/measurements-docker.db
      - ./data/skogsnet-docker.log:/app/skogsnet-docker.log
    restart: unless-stopped
```


## License
MIT License
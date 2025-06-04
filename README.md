# kerio-mirror-go

## Description

kerio-mirror-go is a Go application designed to mirror definition files used by Kerio Control, such as IDS, GeoIP, WebFilter, and Bitdefender databases. It downloads and updates these files on a scheduled basis and serves them via HTTP/HTTPS. It is intended to be used in environments where Kerio Control is deployed, allowing for local access to the latest definitions without relying on external sources.

## Prerequisites

- Go (version 1.18 or higher recommended)

## Setup

1. Clone the repository:

   ```bash
   git clone <repository_url>
   cd kerio-mirror-go
   ```

2. Build the project:

   ```bash
   go build -o kerio-mirror-go ./cmd/server
   ```

3. Create a configuration file (e.g., `config.yaml`) based on the configuration section below.

4. Run the application:

   ```bash
   ./server
   ```

## Run as Windows Service with NSSM

To run the application as a Windows service using NSSM (Non-Sucking Service Manager):

1. Download and install NSSM from [nssm.cc](https://nssm.cc/download).
2. Open a command prompt as Administrator.
3. Run the following command to install the service:
4. `nssm install kerio-mirror-go "C:\path\to\kerio-mirror-go.exe"`
5. Configure the service settings as needed (e.g., startup type, log paths).
6. Start the service:
7. `nssm start kerio-mirror-go`
8. Verify that the service is running:
9. `nssm status kerio-mirror-go`
10. Access the dashboard at `http://localhost`.

## Configuration

The application reads its configuration from a file `config.yaml`.

The configuration file should contain key-value pairs. Based on the code, the following options are expected:

- `DatabasePath`: Path to the SQLite database file.
- `LogPath`: Path to the log file.
- `ScheduleInterval`: Interval (in hours) for scheduled updates.
- `GeoIP4Url`: URL for the GeoIP v4 database.
- `GeoIP6Url`: URL for the GeoIP v6 database.
- `GeoLocUrl`: URL for the GeoLocations file.
- `BitdefenderUrls`: Comma-separated URLs for Bitdefender databases.

Example `config.yaml` file:

```yaml
bitdefender_urls: []
database_path: ./mirror.db
geoip4_url: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv4.csv
geoip6_url: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv6.csv
geoloc_url: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Locations-en.csv
ids_url: ""
license_number: ""
log_path: ./logs/mirror.log
proxy_url: ""
retry_count: 3
retry_delay_seconds: 10
schedule_interval: 23
webfilter_api: https://updates.kerio.com/webfilter/key
```

## Functionality

- **Scheduled Updates**: Downloads and updates definition files at a configurable interval.
- **Database**: Uses SQLite to store information, including the last update time.
- **HTTP/HTTPS Server**: Serves the definition files via HTTP (port 80) and HTTPS (port 443). Requires `cert.pem` and `key.pem` for HTTPS.
- **Supported Definitions**: Mirrors IDS, GeoIP, GeoLocations, WebFilter, and Bitdefender definitions.

## Dependencies

- `github.com/labstack/echo/v4` for the HTTP server.
- `github.com/sirupsen/logrus` for logging.
- `github.com/mattn/go-sqlite3` for SQLite database access.
- `github.com/spf13/viper` for configuration management.
- Other standard Go libraries.

## License

[Specify your license here]
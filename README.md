# kerio-mirror-go

[![Go](https://github.com/TheTitanrain/kerioMirrorGo/actions/workflows/go.yml/badge.svg)](https://github.com/TheTitanrain/kerioMirrorGo/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/TheTitanrain/kerioMirrorGo/branch/main/graph/badge.svg)](https://codecov.io/gh/TheTitanrain/kerioMirrorGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/TheTitanrain/kerioMirrorGo)](https://goreportcard.com/report/github.com/TheTitanrain/kerioMirrorGo)

## Description

kerio-mirror-go is a Go application designed to mirror definition files used by Kerio Control, such as IDS, GeoIP, WebFilter, and Bitdefender databases. It downloads and updates these files on a scheduled basis and serves them via HTTP/HTTPS. It is intended to be used in environments where Kerio Control is deployed, allowing for local access to the latest definitions without relying on external sources.

## Features

- 🔄 **Scheduled Updates**: Automatic daily updates at configured time
- 🗄️ **SQLite Database**: Pure-Go implementation (no CGO required)
- 🌐 **HTTP/HTTPS Server**: Dual-port serving (80/443)
- 🛡️ **Multi-Platform**: Supports Linux, Windows, and macOS
- 📊 **Web Dashboard**: Monitor status and manage settings
- 🔍 **IDS Support**: Versions 1-5 with selective enabling
- 🌍 **GeoIP Mirroring**: IPv4/IPv6 databases
- 🦠 **Bitdefender**: Full mirror + proxy mode with caching
- 🔑 **WebFilter**: License key management
- 🛡️ **Shield Matrix**: On-demand threat data for Kerio Control 9.5+
- 📝 **Snort Template**: IPS template updates for Kerio Control 9.5+
- 📁 **Custom Files**: Mirror any additional URLs
- 🔒 **IP Access Control**: Whitelist/blacklist with CIDR support

## Installation

### Download Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/TheTitanrain/kerioMirrorGo/releases) page:

- `kerio-mirror-go-linux-amd64` - Linux x64
- `kerio-mirror-go-windows-amd64.exe` - Windows x64
- `kerio-mirror-go-darwin-amd64` - macOS x64

### Build from Source

**Prerequisites:**
- Go 1.24.x or higher

**Clone and build:**

```bash
git clone https://github.com/TheTitanrain/kerioMirrorGo.git
cd kerioMirrorGo
go build -o kerio-mirror-go ./cmd/server
```

**Note:** The project uses `modernc.org/sqlite` (pure-Go implementation), so no CGO or GCC is required.

### Run the Application

```bash
./kerio-mirror-go -config config.yaml
```

Default config path is `config.yaml` if not specified.

## Testing

The project includes comprehensive test coverage for key components.

### Run all tests:
```bash
go test ./...
```

### Run tests with verbose output:
```bash
go test ./... -v
```

### Run tests with coverage report:
```bash
go test ./... -cover
```

### Generate HTML coverage report:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run benchmarks:
```bash
go test ./... -bench=. -benchmem
```

### Using Makefile (if make is available):
```bash
make test              # Run all tests
make test-coverage     # Run with coverage
make test-coverage-html # Generate HTML coverage report
make bench             # Run benchmarks
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

The application reads its configuration from `config.yaml`. All settings can also be managed via the web interface at `/settings`.

### Key Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `SCHEDULE_TIME` | Daily update time (HH:MM format) | `03:00` |
| `LICENSE_NUMBER` | Kerio Control license for IDS/WebFilter | Required |
| `DATABASE_PATH` | SQLite database file path | `./mirror.db` |
| `LOG_PATH` | Log file path | `./logs/mirror.log` |
| `PROXY_URL` | HTTP proxy for outbound requests | - |
| `ENABLE_IDS1` - `ENABLE_IDS5` | Enable/disable IDS versions | `true` |
| `ENABLE_BITDEFENDER` | Enable Bitdefender updates | `true` |
| `BITDEFENDER_PROXY_MODE` | Enable proxy mode with caching | `false` |
| `BITDEFENDER_PROXY_BASE_URL` | Upstream URL for proxy mode | `https://upgrade.bitdefender.com` |
| `ENABLE_SHIELD_MATRIX` | Enable Shield Matrix for Kerio 9.5+ | `true` |
| `SHIELD_MATRIX_BASE_URL` | CloudFront base URL for Shield Matrix | `https://d2akeya8d016xi.cloudfront.net/9.5.0` |
| `SHIELD_MATRIX_PRELOAD_FILES` | Preload all Shield Matrix files on schedule | `false` |
| `ENABLE_SNORT_TEMPLATE` | Enable Snort template updates (IDS5) | `true` |
| `SNORT_TEMPLATE_URL` | Snort template download URL | `http://download.kerio.com/control-update/config/v1/snort.tpl` |
| `CUSTOM_DOWNLOAD_URLS` | Array of custom URLs to mirror | `[]` |
| `ALLOWED_IPS` | IP whitelist (CIDR or single IPs) | `[]` |
| `BLOCKED_IPS` | IP blacklist (CIDR or single IPs) | `[]` |
| `RETRY_COUNT` | Download retry attempts | `3` |
| `RETRY_DELAY_SECONDS` | Delay between retries | `10` |

### Example Configuration

```yaml
SCHEDULE_TIME: "03:00"
LICENSE_NUMBER: "your-license-here"
DATABASE_PATH: ./mirror.db
LOG_PATH: ./logs/mirror.log
LOG_LEVEL: info
PROXY_URL: ""

# IDS Settings
ENABLE_IDS1: true
ENABLE_IDS2: true
ENABLE_IDS3: true
ENABLE_IDS4: true
ENABLE_IDS5: true
IDS_URL: https://update.kerio.com/dwn/control/update.php?license=%s&version=%s

# Bitdefender Settings
ENABLE_BITDEFENDER: true
BITDEFENDER_PROXY_MODE: false
BITDEFENDER_PROXY_BASE_URL: https://upgrade.bitdefender.com
BITDEFENDER_URLS: []

# Shield Matrix Settings (Kerio 9.5+)
ENABLE_SHIELD_MATRIX: true
SHIELD_MATRIX_BASE_URL: https://d2akeya8d016xi.cloudfront.net/9.5.0
SHIELD_MATRIX_PRELOAD_FILES: false  # Set to true to preload all files on schedule

# Snort Template Settings
ENABLE_SNORT_TEMPLATE: true
SNORT_TEMPLATE_URL: http://download.kerio.com/control-update/config/v1/snort.tpl

# GeoIP Settings
GEOIP4_URL: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv4.csv
GEOIP6_URL: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Blocks-IPv6.csv
GEOLOC_URL: https://raw.githubusercontent.com/wyot1/GeoLite2-Unwalled/downloads/COUNTRY/CSV/GeoLite2-Country-Locations-en.csv

# WebFilter Settings
WEBFILTER_API: https://updates.kerio.com/webfilter/key

# Custom Downloads
CUSTOM_DOWNLOAD_URLS: []

# IP Access Control
ALLOWED_IPS: []  # Whitelist - if set, only these IPs can access the server
BLOCKED_IPS: []  # Blacklist - these IPs will be blocked (takes priority)

# Retry Settings
RETRY_COUNT: 3
RETRY_DELAY_SECONDS: 10
```

## Usage

### DNS Configuration

To use the mirror server, configure your DNS server to point the following hostnames to your mirror server IP:

**Required DNS entries:**
- `ids-update.kerio.com` → Your mirror server IP
- `update.kerio.com` → Your mirror server IP
- `updates.kerio.com` → Your mirror server IP
- `download.kerio.com` → Your mirror server IP

**For Bitdefender (if enabled):**
- `bdupdate.kerio.com` → Your mirror server IP
- `bda-update.kerio.com` → Your mirror server IP

**For Shield Matrix (Kerio 9.5+):**
- `shieldmatrix-updates.gfikeriocontrol.com` → Your mirror server IP
- `d2akeya8d016xi.cloudfront.net` → Your mirror server IP

**Example DNS configuration:**
```
ids-update.kerio.com.        IN A    192.168.1.100
update.kerio.com.            IN A    192.168.1.100
updates.kerio.com.           IN A    192.168.1.100
download.kerio.com.          IN A    192.168.1.100
bdupdate.kerio.com.          IN A    192.168.1.100
bda-update.kerio.com.        IN A    192.168.1.100
shieldmatrix-updates.gfikeriocontrol.com. IN A 192.168.1.100
d2akeya8d016xi.cloudfront.net. IN A 192.168.1.100
```

Replace `192.168.1.100` with your actual mirror server IP address.

### Web Dashboard

Access the web interface at `http://localhost/` (or `https://localhost/` if HTTPS is configured with `cert.pem` and `key.pem`).

**Available Routes:**
- `/` - Dashboard showing update status and versions
- `/settings` - Configuration management
- `/logs` - View application logs
- `/update.php` - Kerio Control update endpoint
- `/control-update/*` - Serves definition files
- `/getkey.php` - WebFilter key endpoint

### Command Line Options

```bash
./kerio-mirror-go -config /path/to/config.yaml
```

### File Storage

Downloaded files are stored in the `mirror/` directory:
- `mirror/` - IDS files and signatures
- `mirror/bitdefender/` - Bitdefender databases (or cache if proxy mode)
- `mirror/geo/` - GeoIP CSV files
- `mirror/matrix/` - Shield Matrix threat data files (IPv4/IPv6)
- `mirror/custom/` - Custom downloaded files

### Bitdefender Proxy Mode

When `BITDEFENDER_PROXY_MODE: true`, the server acts as a caching proxy:
1. Requests to Bitdefender URLs are forwarded to `BITDEFENDER_PROXY_BASE_URL`
2. Responses are cached locally in `mirror/bitdefender/`
3. Subsequent requests are served from cache
4. Non-cacheable files (versions.id, version.txt, cumulative.txt) are always fetched fresh

### Shield Matrix (Kerio 9.5+)

Shield Matrix provides advanced threat detection for Kerio Control 9.5 and above:

**How it works:**
1. **Version Check**: Periodically checks CloudFront for new Shield Matrix version
2. **Two Download Modes**:
   - **On-Demand** (default): Files downloaded only when Kerio Control requests them
   - **Preload**: All files downloaded on schedule for offline/slow connection environments
3. **Caching**: Downloaded files are cached locally for subsequent requests
4. **File Integrity Check**: When preload mode is enabled, checks for missing files and re-downloads if needed
5. **CloudFront Proxy**: Intercepts CloudFront requests and serves from local cache

**Requests and respones:**
request:
https://shieldmatrix-updates.gfikeriocontrol.com/check_update/?client-id=control&version=9.5.0&last-update=0
response:
{"available": true, "url": "https://d2akeya8d016xi.cloudfront.net/9.5.0/"}

request:
`https://d2akeya8d016xi.cloudfront.net/9.5.0/version`
response:
version file with version content:
1759878869

request:
`https://d2akeya8d016xi.cloudfront.net/9.5.0/ipv4/threat_data_1.dat`
response file:
threat_data_1.dat
...
request:
`https://d2akeya8d016xi.cloudfront.net/9.5.0/ipv4/threat_data_5.dat`
response file:
threat_data_5.dat

request:
`https://d2akeya8d016xi.cloudfront.net/9.5.0/ipv6/threat_data_1.dat`
response file:
threat_data_1.dat
...
request:
`https://d2akeya8d016xi.cloudfront.net/9.5.0/ipv6/threat_data_5.dat`
response file:
threat_data_5.dat

**Supported Files:**
- IPv4: `threat_data_1.dat` to `threat_data_5.dat` (5 files)
- IPv6: `threat_data_1.dat` to `threat_data_5.dat` (5 files)
- Total: 10 files per version

**Configuration:**
```yaml
ENABLE_SHIELD_MATRIX: true
SHIELD_MATRIX_BASE_URL: https://d2akeya8d016xi.cloudfront.net/9.5.0
SHIELD_MATRIX_PRELOAD_FILES: false  # true = preload all files, false = on-demand
```

**Download Modes:**

| Mode | Description | Use Case |
|------|-------------|----------|
| **On-Demand** (`false`) | Files downloaded when requested | Normal internet, minimal storage |
| **Preload** (`true`) | All 10 files downloaded on schedule | Slow/limited internet, offline environments |

**DNS Configuration:**
To use Shield Matrix, configure your DNS to point CloudFront domain to your mirror server:
```
d2akeya8d016xi.cloudfront.net. IN A 192.168.1.100
```

The version number in the URL (9.5.0) corresponds to the Kerio Control version.

### IP Access Control

The application supports IP-based access control with both whitelist and blacklist functionality:

**Features:**
- ✅ Whitelist mode: Only allow specific IPs
- ❌ Blacklist mode: Block specific IPs
- 🌐 CIDR notation support (e.g., `192.168.1.0/24`)
- 📍 Single IP support (e.g., `192.168.1.100`)
- 🔍 Automatic IP detection from headers (`X-Real-IP`, `X-Forwarded-For`)
- 📝 Detailed logging of blocked access attempts

**Configuration:**
```yaml
ALLOWED_IPS:
  - 192.168.1.100      # Single IP
  - 192.168.2.0/24     # CIDR range
  - 10.0.0.0/8         # Large CIDR range

BLOCKED_IPS:
  - 203.0.113.50       # Block specific IP
  - 198.51.100.0/24    # Block entire range
```

**Access Logic:**
1. If IP is in `BLOCKED_IPS` → **403 Forbidden** (blacklist takes priority)
2. If `ALLOWED_IPS` is set and IP is NOT in list → **403 Forbidden**
3. If both lists are empty → **Allow all** (no restrictions)

**Web Configuration:**
IP access control can be configured via the web interface at `/settings` under the "IP Access Control" section. Enter one IP or CIDR range per line.

**Logging:**
All blocked access attempts are logged with warning level:
```
WARN[0123] IP filter: blocked IP 203.0.113.50 attempting to access /update.php
WARN[0124] IP filter: unauthorized IP 1.2.3.4 attempting to access /
```

## API Endpoints

### Update Endpoint (Kerio Control)

```http
GET /update.php?license=XXX&version=9.4.1
```

Returns update information for Kerio Control, mimicking the official API.

### WebFilter Key

```http
GET /getkey.php?number=YOUR_LICENSE
```

Returns the WebFilter key for the specified license number.

## Development

### Project Structure

```
kerioMirrorGo/
├── cmd/server/          # Main application entry point
├── config/              # Configuration management
├── db/                  # Database initialization and schema
├── handlers/            # HTTP request handlers
├── logging/             # Logging utilities
├── middleware/          # HTTP middleware (IP filtering, etc.)
│   └── ipfilter.go      # IP access control middleware
├── mirror/              # Mirror logic for each component
│   ├── bitdefender.go
│   ├── bitdefender_proxy.go
│   ├── custom.go
│   ├── geo.go
│   ├── ids.go
│   ├── mirror.go
│   ├── shieldmatrix.go  # Shield Matrix (Kerio 9.5+)
│   ├── snort.go         # Snort template
│   └── webfilter.go
├── utils/               # Utilities (HTTP client, file ops)
├── templates/           # HTML templates (embedded)
└── static/              # Static assets (embedded)
```

### Dependencies

- **Web Framework**: `github.com/labstack/echo/v4`
- **Logging**: `github.com/sirupsen/logrus`
- **Database**: `modernc.org/sqlite` (pure-Go, no CGO)
- **Configuration**: `github.com/spf13/viper`

### CI/CD

The project uses GitHub Actions for:
- Multi-platform builds (Linux, Windows, macOS)
- Automated testing with coverage reports
- Automatic release creation on version tags
- Binary artifact uploads

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `go test ./...` to verify
5. Submit a pull request

## License

[Specify your license here]

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/TheTitanrain/kerioMirrorGo).
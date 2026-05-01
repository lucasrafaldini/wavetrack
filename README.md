# WaveTrack

*Read this in other languages: [Português](README.pt-BR.md)*

Presence monitoring system based on Wi-Fi device detection. WaveTrack captures device handshakes on a Wi-Fi network to automatically record employee presence.

> **[Quick Start Guide](.github/docs/QUICKSTART.md)** | [Design Document](.github/docs/SDD.md) | [Deploy Guide](.github/docs/DEPLOY.md) | [Contributing](.github/docs/CONTRIBUTING.md) | [Changelog](.github/docs/CHANGELOG.md)

## Features

### Monitoring System
- **Wi-Fi Monitoring**: Captures Wi-Fi management packets to detect nearby devices
- **Presence Registration**: Associates devices (MACs) with registered employees
- **Automatic Events**: Automatically detects arrivals and departures
- **Logging System**: Logs all events in daily rotated JSON files
- **SQLite Database**: Permanent presence history for analysis and reports

### Complete Web Interface
- **Real-Time Dashboard**: View active devices and their information
- **Tab System**:
  - **Real Time**: Monitor currently connected devices
  - **History (7 days)**: Analyze presence over the last 7 days with percentages
  - **Employees**: Manage employee registrations (create, edit, delete)
- **Dynamic Registration**: Add and edit employees directly through the interface
- **Sorting and Filters**: Organize data by name, online time, arrival time, etc.

### Employee Management
- **Full CRUD**: Create, view, edit, and delete employees
- **Editing with History Preservation**:
  - Change MAC address and all history is automatically migrated
  - Update name and past events reflect the change
  - Edit department without affecting history
- **Smart Deletion**: Remove employees while keeping 100% of the history for reports

### History and Reports
- **7-Day History**: View daily presence with:
  - Arrival and departure times
  - Total hours worked
  - Presence percentage (base: 8 hours = 100%)
  - Weekly presence average
- **Preserved Data**: History is never deleted, even when removing employees
- **REST API**: Complete endpoints for integration with other systems

## Requirements

### Operating System
- **macOS** or **Linux** with Wi-Fi interface
- Administrator privileges (required for packet capture)

### Dependencies
- Go 1.21 or higher
- libpcap (for packet capture)

#### libpcap Installation

**macOS:**
```bash
brew install libpcap
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install libpcap-dev
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install libpcap-devel
```

## Installation

1. **Clone the repository:**
```bash
git clone https://github.com/lucasrafaldini/wavetrack.git
cd wavetrack
```

2. **Install dependencies:**
```bash
go mod download
```

3. **Configure the project:**
  - Edit `config.yaml` with your settings
  - Adjust the network interface (usually `en0` on macOS, `wlan0` on Linux)
  - Employees will be registered via the web interface after starting the system

4. **Build the project:**
```bash
go build -o wavetrack cmd/wavetrack/main.go
```

5. **Adjust database permissions (macOS):**

If the database is created with `sudo`, you'll need to adjust permissions to allow non-root access:

```bash
# After the first run, adjust the data directory owner
sudo chown -R $(whoami):staff ./data

# Or run the app without sudo, but with capture permissions:
# (requires additional system configuration)
```

## Configuration

Edit the `config.yaml` file:

```yaml
network:
  interface: "en0" # Wi-Fi network interface
  channel: 6 # Wi-Fi channel
  scan_interval: 10 # Scan interval in seconds

logging:
  log_dir: "./logs"
  log_level: "info"

presence:
  timeout_minutes: 20 # Inactivity time before marking offline (min. 20 min)
  signal_threshold: -75 # Minimum signal in dBm (0 = ignore; used in non-monitor mode)
```

**Note**: Employees are now registered via the web interface and stored in the SQLite database (`data/wavetrack.db`). There is no longer an employees section in `config.yaml`.

### How to find a device's MAC address:

**iPhone/iPad:**
- Settings → General → About → Wi-Fi Address

**Android:**
- Settings → About phone → Status → Wi-Fi MAC address

**Laptop:**
```bash
# macOS
ifconfig en0 | grep ether

# Linux
ip link show wlan0
```

## REST API

WaveTrack exposes a REST API for integration with other systems:

### Available Endpoints:

#### `GET /api/devices`
Lists all detected devices

**Response:**
```json
[
  {
    "mac_address": "00:11:22:33:44:55",
    "type": "smartphone",
    "vendor": "Apple",
    "signal_strength": -45,
    "first_seen": "2025-10-28T14:30:00Z",
    "last_seen": "2025-10-28T15:45:00Z",
    "is_active": true,
    "employee_name": "João Silva",
    "department": "TI",
    "first_seen_today": "2025-10-28T08:15:00Z",
    "online_duration_seconds": 14400
  }
]
```

#### `GET /api/employees`
Lists all registered employees

**Response:**
```json
[
  {
    "id": 1,
    "mac_address": "00:11:22:33:44:55",
    "name": "João Silva",
    "department": "TI",
    "custom_device_type": "iPhone",
    "custom_vendor": "Apple",
    "created_at": "2025-10-28T10:00:00Z",
    "updated_at": "2025-10-28T10:00:00Z"
  }
]
```

#### `POST /api/associate`
Creates or updates an employee

**Request (New):**
```json
{
  "mac_address": "00:11:22:33:44:55",
  "name": "João Silva",
  "department": "TI",
  "custom_device_type": "iPhone 15 Pro",
  "custom_vendor": "Apple"
}
```

**Request (Edit with MAC change):**
```json
{
  "mac_address": "00:11:22:33:44:66",
  "old_mac_address": "00:11:22:33:44:55",
  "name": "João Silva",
  "department": "TI",
  "custom_device_type": "iPhone 15 Pro",
  "custom_vendor": "Apple"
}
```

**Response:**
```json
{
  "message": "Employee registered successfully"
}
```

**Note:** When changing the MAC address, all event history is automatically migrated to the new MAC.

#### `DELETE /api/employees/{mac}`
Removes an employee (preserves history)

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/employees/00:11:22:33:44:55
```

**Response:**
```json
{
  "message": "Employee deleted successfully"
}
```

**Note:** The presence history is preserved for reports and analysis.

#### `GET /api/history/7days`
Returns presence history for the last 7 days

**Response:**
```json
[
  {
    "date": "2025-10-28",
    "day_of_week": "Monday",
    "mac_address": "00:11:22:33:44:55",
    "name": "João Silva",
    "department": "TI",
    "arrival": "08:15:23",
    "departure": "18:30:45",
    "total_hours": "10.25",
    "percentage": "128%"
  }
]
```

#### `GET /api/stats`
Returns system statistics

**Response:**
```json
{
  "total_devices": 15,
  "active_devices": 8,
  "total_employees": 5,
  "events_today": 24,
  "offline_threshold_minutes": 20
}
```

## Usage

**Run with administrator privileges:**

```bash
# macOS
sudo ./wavetrack

# Linux
sudo ./wavetrack

# With a custom port for the web server (default: 8080)
sudo ./wavetrack -port 3000

# With a custom configuration file
sudo ./wavetrack -config /path/to/config.yaml
```

### Accessing the Web Interface

After starting the system, open your browser and access:

```
http://localhost:8080
```

### Interface Features:

#### 1. **Real Time (Main Tab)**
  - View all devices detected at the moment
  - Activity status (Active/Inactive) with a 20-minute threshold
  - Real-time signal strength (N/A in non-monitor mode)
  - First arrival of the day ("Arrived at")
  - Total online time for the current day
  - Real-time system statistics

#### 2. **7-Day History**
  - Complete view of the last 7 days of presence
  - For each employee and day:
    - Date and day of the week
    - Arrival time (first arrival)
    - Departure time (last departure)
    - Total hours worked
    - Presence percentage (8 hours = 100%)
    - Weekly presence average per employee
  - Data loaded from SQLite database

#### 3. **Employee Management**
  - **List**: View all registered employees
  - **Create**: Add new employees with name, MAC, department
  - **Edit**: Modify existing employee data
    - Change MAC: History is automatically migrated
    - Change Name: All events are updated
    - Change Department: Record updated
  - **Delete**: Remove employees (with confirmation)
    - Employee is removed from the active list
    - All history is preserved for reports

#### 4. **Sorting and Filters**
  - Sort by: Employee name, Arrived at, Online time
  - Filter: Employees only (hides unregistered devices)
  - Direction: Ascending or Descending

### Expected Output (Console):
```
=== WaveTrack - Presence Monitoring System ===
Configuration loaded: interface=en0, interval=10s
Storage system started
Logging system started: ./logs/presence_2025-10-28.log
Scanner started on interface en0
Monitoring 2 registered employees
  Web server started on http://localhost:8080
  Access the dashboard in the browser!
System started! Press Ctrl+C to stop...
  João Silva arrived (Department: TI)
  Maria Santos arrived (Department: HR)
```

## Project Structure

```
wavetrack/
├── cmd/
│   ├── wavetrack/
│   │   └── main.go           # Main application
│   └── seed_history/
│       └── main.go           # Historical data generator (dev)
├── internal/
│   ├── api/
│   │   └── handlers.go       # REST API endpoints
│   ├── config/
│   │   └── config.go         # Configuration management
│   ├── deviceid/
│   │   └── deviceid.go       # Vendor/type identification via OUI
│   ├── logger/
│   │   └── logger.go         # Logging system
│   ├── models/
│   │   └── models.go         # Data models
│   ├── storage/
│   │   ├── storage.go        # SQLite persistence
│   │   ├── schema.sql        # DB schema
│   │   └── storage_test.go   # Storage tests
│   ├── tracker/
│   │   └── tracker.go        # Tracking logic
│   └── wifi/
│       └── scanner.go        # Wi-Fi packet capture
├── web/
│   ├── index.html            # Web interface (structure)
│   ├── css/
│   │   └── style.css         # Interface styles
│   └── js/
│       └── app.js            # JavaScript logic
├── data/                     # SQLite database (created automatically)
│   └── wavetrack.db
├── logs/                     # Log files (created automatically)
├── config.yaml               # Configuration file
├── migrate_preserve_history.sql # Database migration (applied)
├── go.mod                    # Go dependencies
└── README.md                 # This file
```

## Log Format

Events are saved in JSON format rotated daily (`logs/presence_YYYY-MM-DD.log`):

```json
{
  "timestamp": "2025-10-28T14:30:00Z",
  "event_type": "arrival",
  "employee_name": "João Silva",
  "mac_address": "00:11:22:33:44:55",
  "signal_strength": -45,
  "metadata": ""
}
```

**Event types:**
- `arrival`: Employee arrived
- `departure`: Employee left
- `unknown_device`: Unknown device detected

## Important Considerations

1. **Permissions**: The program must run as root/sudo to capture packets
2. **Database Permissions (macOS)**: If the SQLite DB is created with sudo, adjust the owner with `sudo chown -R $(whoami):staff ./data` after the first run
3. **Monitor Mode vs Compatibility**: On macOS, the scanner uses a compatible ARP/Ethernet mode when 802.11 is not available. Signal strength (RSSI) is not captured in this mode
4. **Privacy**: This system captures MAC addresses. Make sure you comply with local privacy laws (LGPD, GDPR, etc.)
5. **Range**: Detection depends on the Wi-Fi signal range (typically 10-50 meters)
6. **Offline Threshold**: Devices are marked offline after 20 minutes of inactivity (configurable in `presence.timeout_minutes`, minimum 20 minutes)
7. **Preserved History**: When deleting an employee, all presence history is kept in the database for reporting and auditing purposes
8. **Data Migration**: When editing an employee's MAC, all history is automatically migrated to the new MAC

## Documentation

- **[SDD.md](.github/docs/SDD.md)** - Software Design Document (architecture, components, design decisions)
- **[QUICKSTART.md](.github/docs/QUICKSTART.md)** - Quick start guide
- **[DEPLOY.md](.github/docs/DEPLOY.md)** - Deployment guide
- **[CONTRIBUTING.md](.github/docs/CONTRIBUTING.md)** - How to contribute to the project
- **[CHANGELOG.md](.github/docs/CHANGELOG.md)** - Change history
- **[ROADMAP.md](.github/docs/ROADMAP.md)** - Development roadmap

## Roadmap

### Implemented (v1.0)
- [x] Web interface for real-time visualization
- [x] Manager dashboard with tab system
- [x] Dynamic employee registration via web (full CRUD)
- [x] REST API for integration
- [x] SQLite database for presence history
- [x] Online time and first arrival of the day calculation
- [x] Device sorting and filtering on the dashboard
- [x] 7-day history with presence percentages
- [x] Employee editing with automatic history migration
- [x] History preservation when deleting employees
- [x] Tab system (Real Time, History, Employees)
- [x] Code organization (separated HTML, CSS, JS)

### Implemented (v2.0)
- [x] **Integration with OUIja** - Custom library for MAC Address lookup
  - Official IEEE database via Wireshark (38k+ OUIs)
  - Smart in-memory cache with sync.RWMutex
  - Automatic database update (7-day TTL)
  - API endpoints: vendor details, search, top vendors, stats
- [x] **Removal of hardcoded OUI database** - Replaced by OUIja
- [x] **CI/CD with GitHub Actions** - Lint, tests with coverage, benchmark, build matrix
- [x] **OUIja Endpoints in REST API** - `/api/vendor/details`, `/search`, `/top`, `/stats`

### Upcoming Features

#### v2.1 - Interface and UX
- [ ] Responsive web interface (mobile-first)
- [ ] Enhanced real-time dashboard with interactive charts
- [ ] Push notification system

#### v2.2 - Enterprise Features
- [ ] Daily presence report (PDF/CSV)
- [ ] Webhook notifications (email/Slack)
- [ ] Support for multiple devices per employee
- [ ] Authentication and access control
- [ ] HR systems integration

> **Full Roadmap**: See [ROADMAP.md](.github/docs/ROADMAP.md) for technical details and the full schedule

## License

MIT License

## Author

Lucas Rafaldini (@lucasrafaldini)

---

**Version:** 2.0  
**Status:** Production Ready  
**Last Updated:** April 2026  

**Note**: Full system with web interface, 7-day history, employee CRUD, smart historical data preservation, and device identification via OUIja (official IEEE database).
# picoHWMon - Cross-Platform System Monitoring & Control

A modular, cross-platform system monitoring and control application built in Go with REST API support.

## 🌍 Platform Support

- **Linux** (Debian/Ubuntu/Arch)
- **Windows 10+**

## 📦 Architecture

The application uses a modular Go architecture with platform-specific implementations:

```
├── main.go                 # Main application entry point
├── go.mod                  # Go module dependencies
├── api/                    # REST API layer (Fiber)
│   ├── server.go          # API server setup
│   └── handlers.go        # HTTP handlers
└── internal/              # Internal packages
    ├── platform/          # Platform detection
    ├── cpu/               # CPU monitoring
    ├── gpu/               # GPU monitoring & control
    ├── memory/            # Memory monitoring
    ├── disk/              # Disk monitoring
    ├── temps/             # Temperature monitoring
    ├── fan/               # Fan control
    └── overclock/         # Overclocking control
```

### Build Tags

Platform-specific files use Go build tags:
- `*_linux.go` → `//go:build linux`
- `*_windows.go` → `//go:build windows`

## 🌐 REST API Endpoints

The application exposes system data and control endpoints via Fiber:

### System Information (GET)
- `GET /api/cpu` - CPU model, cores, threads, usage %
- `GET /api/gpu` - GPU vendor, model, VRAM, usage %
- `GET /api/memory` - RAM total, used, available
- `GET /api/disk` - Disk usage for all mounted drives
- `GET /api/temps` - CPU, GPU, and system temperatures

### Control Endpoints (POST)
- `POST /api/fan/:id/settings` - Set fan speed (auto, fixed, or curve mode)
- `GET /api/gpu/:id/overclock` - Get GPU overclock settings
- `POST /api/gpu/:id/overclock` - Set GPU overclock settings
- `GET /api/overclock/profiles` - Get saved overclock profiles
- `POST /api/overclock/profiles` - Save overclock profile
- `POST /api/overclock/profiles/:name/load` - Load overclock profile

### Health Check
- `GET /api/health` - Service health and platform info

### 📚 Complete API Documentation

- **[Full API Documentation](API_DOCUMENTATION.md)** - Comprehensive guide with all endpoints, parameters, and examples
- **[Quick Reference](API_QUICK_REFERENCE.md)** - Cheat sheet for common API operations
- **[Diagnostic Script](diagnose.sh)** - Hardware detection troubleshooting tool

### 🆕 New Features

- **Fan Curve Mode**: Temperature-based automatic fan control with custom curves
- **AMD GPU Support**: Full AMD GPU detection and basic overclocking support
- **Enhanced CPU Detection**: Proper core/thread count for Intel hyperthreading CPUs
- **Real-time Monitoring**: Live temperature monitoring with 2-second update intervals

## 🛠️ Dependencies

### Linux Requirements

```bash
# Ubuntu/Debian
sudo apt-get install lm-sensors

# Arch Linux
sudo pacman -S lm_sensors

# NVIDIA GPU support (optional)
nvidia-smi

# AMD GPU support (optional)
rocm-smi
```

### Windows Requirements

- Windows 10 or later
- NVIDIA drivers (for NVIDIA GPU support)
- AMD drivers (for AMD GPU support)

## 🚀 Installation & Usage

### Build from Source

```bash
# Clone the repository
git clone https://github.com/CristiGvl/picoHWMon.git
cd picoHWMon

# Install dependencies
go mod tidy

# Build the application
go build -o picoHWMon main.go

# Run the application
./picoHWMon --port 8080
```

### Cross-Platform Build

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o picoHWMon-linux main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o picoHWMon-windows.exe main.go
```

### Docker (Linux only)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o picoHWMon main.go

FROM alpine:latest
RUN apk --no-cache add lm-sensors
WORKDIR /root/
COPY --from=builder /app/picoHWMon .
EXPOSE 8080
CMD ["./picoHWMon"]
```

## 📊 API Examples

### Get CPU Information
```bash
curl http://localhost:8080/api/cpu
```

Response:
```json
{
  "model": "Intel(R) Core(TM) i7-10700K CPU @ 3.80GHz",
  "cores": 8,
  "threads": 16,
  "usage_percent": 25.5,
  "frequency_mhz": 3800
}
```

### Get GPU Information
```bash
curl http://localhost:8080/api/gpu
```

Response:
```json
[
  {
    "vendor": "nvidia",
    "model": "GeForce RTX 3080",
    "vram_mb": 10240,
    "usage_percent": 45.2,
    "memory_usage_percent": 60.1,
    "temperature_celsius": 72.0,
    "power_usage_watts": 285.5,
    "clock_core_mhz": 1905,
    "clock_memory_mhz": 9751
  }
]
```

### Set Fan Speed
```bash
curl -X POST http://localhost:8080/api/fan/0/settings \
  -H "Content-Type: application/json" \
  -d '{
    "mode": "fixed",
    "fixed_speed_percent": 75
  }'
```

### Set GPU Overclock
```bash
curl -X POST http://localhost:8080/api/gpu/0/overclock \
  -H "Content-Type: application/json" \
  -d '{
    "core_clock_offset_mhz": 100,
    "memory_clock_offset_mhz": 500,
    "power_limit_percent": 120,
    "fan_speed_percent": 80
  }'
```

## ⚙️ Platform-Specific Implementation

### Linux Implementation

- **CPU/RAM/Disk**: Uses `gopsutil` library
- **Temperatures**: `lm-sensors` integration
- **GPU Support**:
  - NVIDIA: `nvidia-smi` command-line tool
  - AMD: `rocm-smi` or `/sys/class/drm/` filesystem
- **Fan Control**: `pwmconfig` and `fancontrol` integration
- **Overclocking**: NVIDIA via `nvidia-smi`, AMD via sysfs or `rocm-smi`

### Windows Implementation

- **System Info**: WMI/CIM queries
  - CPU: `Win32_Processor`
  - Memory: `Win32_PhysicalMemory`
  - Disk: `Win32_LogicalDisk`
- **GPU Support**:
  - NVIDIA: `nvidia-smi` command-line tool
  - AMD: WMI queries for detection, OverdriveNTool for overclocking
- **Temperatures**: WMI sensors or OpenHardwareMonitor
- **Overclocking**: 
  - NVIDIA: Power limit control via `nvidia-smi` (clock offsets require MSI Afterburner/EVGA Precision)
  - AMD: Full overclocking support via `OverdriveNTool.exe`

## 🔧 Configuration

The application automatically detects the platform and uses appropriate monitoring methods. No configuration file is required for basic operation.

### Command-Line Options

```bash
./picoHWMon --help

Usage:
  -port string
        Port to run the server on (default "8080")
```

## 🧪 Development

### Project Structure

```bash
# Run tests
go test ./...

# Run with race detection
go run -race main.go

# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Adding New Modules

1. Create interface in `internal/newmodule/newmodule.go`
2. Implement platform-specific files with build tags
3. Add API handlers in `api/handlers.go`
4. Update server routes in `api/server.go`

## 📝 Limitations & Future Enhancements

### Current Limitations

- Fan control implementation is basic (platform-dependent)
- NVIDIA overclocking on Windows limited to power limits (requires additional tools for clocks)
- AMD overclocking on Windows requires OverdriveNTool.exe
- Some sensors may require elevated privileges
- AMD GPU support varies by platform

### Planned Features

- [ ] WebSocket support for real-time monitoring
- [ ] Web dashboard UI
- [ ] Advanced fan curve configuration
- [ ] Enhanced NVIDIA overclocking integration (MSI Afterburner SDK)
- [ ] System alerts and notifications
- [ ] Historical data logging
- [ ] macOS support

## 🔒 Security Considerations

- Run with appropriate privileges for hardware access
- API endpoints for control operations should be secured in production
- Overclocking features can damage hardware if misused
- Windows NVIDIA overclocking requires additional tools for full functionality
- AMD Windows overclocking requires OverdriveNTool.exe to be installed
- Consider rate limiting for API endpoints

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 🐛 Troubleshooting

### Linux Issues

```bash
# If lm-sensors not working
sudo sensors-detect

# For permission issues
sudo usermod -a -G adm $USER

# NVIDIA GPU not detected
nvidia-smi

# AMD overclocking on Windows (requires OverdriveNTool)
# Download from: https://forums.guru3d.com/threads/overdriventool-tool-for-amd-gpus.416116/
# Place OverdriveNTool.exe in your PATH
```

### Windows Issues

- Ensure WMI service is running
- Run as Administrator for full hardware access
- Check Windows version compatibility

For more issues, please check the [Issues](https://github.com/CristiGvl/picoHWMon/issues) page.

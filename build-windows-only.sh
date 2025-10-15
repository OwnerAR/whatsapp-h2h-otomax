#!/bin/bash
# WhatsApp H2H Otomax - Windows Only Build Script
# Optimized for Windows cross-compilation from Linux/WSL

set -e

echo "========================================"
echo "WhatsApp H2H Otomax - Windows Builder"
echo "========================================"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed or not in PATH"
    echo "Please install Go 1.24.0 or higher from https://go.dev/dl/"
    exit 1
fi

echo "[1/5] Checking Go version..."
go version

echo
echo "[2/5] Setting up Windows cross-compilation..."

# Check and install mingw-w64 if needed
if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "Installing mingw-w64 for Windows cross-compilation..."
    
    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y gcc-mingw-w64
    elif command -v yum &> /dev/null; then
        sudo yum install -y mingw64-gcc
    elif command -v pacman &> /dev/null; then
        sudo pacman -S mingw-w64-gcc
    elif command -v brew &> /dev/null; then
        brew install mingw-w64
    else
        echo "ERROR: Cannot install mingw-w64 automatically."
        echo "Please install mingw-w64 manually for your system."
        echo "Ubuntu/Debian: sudo apt-get install gcc-mingw-w64"
        echo "CentOS/RHEL: sudo yum install mingw64-gcc"
        echo "Arch: sudo pacman -S mingw-w64-gcc"
        exit 1
    fi
fi

echo "Windows cross-compilation tools ready."

echo
echo "[3/5] Downloading dependencies..."
go mod download

echo
echo "[4/5] Building for Windows..."

# Set environment for Windows cross-compilation
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc

# Create build directory
mkdir -p build/windows

# Build with optimizations
echo "Building Windows executable..."
go build -ldflags="-s -w" -o build/windows/whatsapp-h2h.exe cmd/server/main.go

echo
echo "[5/5] Creating Windows deployment package..."

# Copy configuration files
cp .env.example build/windows/ 2>/dev/null || true
cp README.md build/windows/ 2>/dev/null || true
cp MIGRATION_NOTES.md build/windows/ 2>/dev/null || true

# Create directories
mkdir -p build/windows/db
mkdir -p build/windows/logs

# Create Windows batch files
cat > build/windows/start.bat << 'EOF'
@echo off
title WhatsApp H2H Otomax
echo ========================================
echo WhatsApp H2H Otomax - Starting...
echo ========================================
echo.
echo Make sure you have configured the .env file!
echo.
whatsapp-h2h.exe
echo.
echo Application stopped. Press any key to exit...
pause >nul
EOF

cat > build/windows/install-service.bat << 'EOF'
@echo off
title WhatsApp H2H Service Installer
echo ========================================
echo WhatsApp H2H Otomax - Service Installer
echo ========================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script must be run as Administrator!
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo Installing WhatsApp H2H as Windows Service...
echo.

REM Get the full path to the executable
set "SERVICE_PATH=%~dp0whatsapp-h2h.exe"
set "SERVICE_PATH=%SERVICE_PATH:\=\\%"

REM Create the service
sc create "WhatsAppH2H" binPath= "%SERVICE_PATH%" start= auto DisplayName= "WhatsApp H2H Otomax"
if %errorlevel% neq 0 (
    echo ERROR: Failed to create service
    pause
    exit /b 1
)

REM Set service description
sc description "WhatsAppH2H" "WhatsApp Host-to-Host middleware for Otomax integration"

echo.
echo Service installed successfully!
echo.
echo To start the service: sc start WhatsAppH2H
echo To stop the service:  sc stop WhatsAppH2H
echo To delete the service: sc delete WhatsAppH2H
echo.
echo Press any key to exit...
pause >nul
EOF

cat > build/windows/uninstall-service.bat << 'EOF'
@echo off
title WhatsApp H2H Service Uninstaller
echo ========================================
echo WhatsApp H2H Otomax - Service Uninstaller
echo ========================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script must be run as Administrator!
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo Uninstalling WhatsApp H2H Service...
echo.

REM Stop the service first
sc stop "WhatsAppH2H" >nul 2>&1
echo Service stopped.

REM Delete the service
sc delete "WhatsAppH2H"
if %errorlevel% neq 0 (
    echo ERROR: Failed to delete service
    pause
    exit /b 1
)

echo.
echo Service uninstalled successfully!
echo.
echo Press any key to exit...
pause >nul
EOF

cat > build/windows/start-service.bat << 'EOF'
@echo off
title WhatsApp H2H Service Controller
echo ========================================
echo WhatsApp H2H Otomax - Service Controller
echo ========================================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script must be run as Administrator!
    echo Right-click and select "Run as administrator"
    pause
    exit /b 1
)

echo Starting WhatsApp H2H Service...
sc start "WhatsAppH2H"
if %errorlevel% neq 0 (
    echo ERROR: Failed to start service
    echo Make sure the service is installed first!
    pause
    exit /b 1
)

echo.
echo Service started successfully!
echo.
echo To check service status: sc query WhatsAppH2H
echo To stop the service: sc stop WhatsAppH2H
echo.
echo Press any key to exit...
pause >nul
EOF

# Create PowerShell script for advanced users
cat > build/windows/start.ps1 << 'EOF'
# WhatsApp H2H Otomax - PowerShell Launcher
param(
    [switch]$Service,
    [switch]$Install,
    [switch]$Uninstall
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "WhatsApp H2H Otomax - PowerShell Launcher" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if ($Install) {
    Write-Host "Installing as Windows Service..." -ForegroundColor Yellow
    $servicePath = (Resolve-Path ".\whatsapp-h2h.exe").Path
    sc.exe create "WhatsAppH2H" binPath= $servicePath start= auto
    sc.exe description "WhatsAppH2H" "WhatsApp Host-to-Host middleware for Otomax"
    Write-Host "Service installed successfully!" -ForegroundColor Green
    return
}

if ($Uninstall) {
    Write-Host "Uninstalling Windows Service..." -ForegroundColor Yellow
    sc.exe stop "WhatsAppH2H"
    sc.exe delete "WhatsAppH2H"
    Write-Host "Service uninstalled successfully!" -ForegroundColor Green
    return
}

if ($Service) {
    Write-Host "Starting as Windows Service..." -ForegroundColor Yellow
    sc.exe start "WhatsAppH2H"
    Write-Host "Service started!" -ForegroundColor Green
    return
}

Write-Host "Starting WhatsApp H2H Otomax..." -ForegroundColor Green
Write-Host "Press Ctrl+C to stop the application" -ForegroundColor Yellow
Write-Host ""

# Start the application
& ".\whatsapp-h2h.exe"
EOF

# Create a comprehensive README for Windows
cat > build/windows/README-WINDOWS.md << 'EOF'
# WhatsApp H2H Otomax - Windows Installation Guide

## Quick Start

1. **Configure the application:**
   - Edit `whatsapp-h2h.env` with your settings
   - Set your API key, webhook URL, and other configurations

2. **Run the application:**
   - Double-click `start.bat` to run directly
   - Or use PowerShell: `.\start.ps1`

## Windows Service Installation

### Install as Service
1. Right-click `install-service.bat` → "Run as administrator"
2. The service will be installed and set to start automatically

### Start/Stop Service
- **Start:** Right-click `start-service.bat` → "Run as administrator"
- **Stop:** Open Command Prompt as Administrator → `sc stop WhatsAppH2H`
- **Status:** `sc query WhatsAppH2H`

### Uninstall Service
1. Right-click `uninstall-service.bat` → "Run as administrator"

## PowerShell Usage

```powershell
# Run directly
.\start.ps1

# Install as service
.\start.ps1 -Install

# Start service
.\start.ps1 -Service

# Uninstall service
.\start.ps1 -Uninstall
```

## Configuration

Edit the `whatsapp-h2h.env` file:

```env
# Server Configuration
PORT=8080
HOST=0.0.0.0

# WhatsApp Configuration
WA_DB_PATH=./db/whatsmeow.db
WA_LOG_LEVEL=INFO

# Otomax Webhook
OTOMAX_WEBHOOK_URL=https://your-otomax.com/api/webhook/whatsapp
OTOMAX_WEBHOOK_TIMEOUT=10s
OTOMAX_WEBHOOK_RETRY_COUNT=3

# Security
API_KEY=your-secret-api-key

# Message Tracking
MESSAGE_TRACKING_TTL=24h
TRACKING_DB_PATH=./db/tracking.db

# Webhook Whitelist (optional)
WEBHOOK_WHITELIST_JIDS=
```

## First Run

1. Start the application using any method above
2. Look for `whatsapp-qrcode.png` in the application directory
3. Open WhatsApp on your phone: Settings → Linked Devices → Link a Device
4. Scan the QR code
5. The application will connect and start the HTTP server
6. Test the connection: `curl http://localhost:8080/health`

## Troubleshooting

### Antivirus Issues
- Add the application folder to your antivirus exclusions
- Some antiviruses may flag the executable as suspicious

### Firewall Issues
- Allow the application through Windows Firewall
- Port 8080 should be accessible

### Service Issues
- Check service status: `sc query WhatsAppH2H`
- View service logs in Event Viewer
- Ensure the service is running as the correct user

### Port Already in Use
- Check if port 8080 is in use: `netstat -an | findstr :8080`
- Change the PORT in the .env file if needed

## File Structure

```
windows/
├── whatsapp-h2h.exe          # Main executable
├── whatsapp-h2h.env          # Configuration file
├── start.bat                 # Direct execution
├── install-service.bat      # Install as Windows service
├── uninstall-service.bat    # Uninstall service
├── start-service.bat         # Start the service
├── start.ps1                # PowerShell launcher
├── db/                       # Database directory
├── logs/                     # Log files directory
└── README-WINDOWS.md         # This file
```

## Support

For issues and support, check the main README.md file or contact your system administrator.
EOF

echo
echo "========================================"
echo "WINDOWS BUILD COMPLETED SUCCESSFULLY!"
echo "========================================"
echo
echo "Output directory: build/windows/"
echo "Executable: build/windows/whatsapp-h2h.exe"
echo
echo "Files created:"
echo "├── whatsapp-h2h.exe          # Main executable"
echo "├── start.bat                 # Direct execution"
echo "├── install-service.bat       # Install as Windows service"
echo "├── uninstall-service.bat     # Uninstall service"
echo "├── start-service.bat         # Start the service"
echo "├── start.ps1                 # PowerShell launcher"
echo "├── README-WINDOWS.md         # Windows installation guide"
echo "├── db/                       # Database directory"
echo "└── logs/                     # Log files directory"
echo
echo "File size:"
ls -lh build/windows/whatsapp-h2h.exe 2>/dev/null || echo "  whatsapp-h2h.exe: N/A"
echo
echo "Next steps:"
echo "1. Copy build/windows/ to your Windows machine"
echo "2. Edit whatsapp-h2h.env with your configuration"
echo "3. Run start.bat to start the application"
echo "4. Or run install-service.bat as Administrator to install as service"
echo
echo "For detailed instructions, see build/windows/README-WINDOWS.md"

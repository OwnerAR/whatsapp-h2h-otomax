#!/bin/bash
# WhatsApp H2H Otomax - Cross-Platform Build Script
# Builds for Windows, Linux, and macOS from any platform

set -e

echo "========================================"
echo "WhatsApp H2H Otomax - Cross-Platform Builder"
echo "========================================"
echo

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed or not in PATH"
    echo "Please install Go 1.24.0 or higher from https://go.dev/dl/"
    exit 1
fi

echo "[1/6] Checking Go version..."
go version

echo
echo "[2/6] Setting up cross-compilation environment..."

# Function to check if cross-compilation tools are available
check_windows_tools() {
    if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
        echo "WARNING: x86_64-w64-mingw32-gcc not found."
        echo "Installing mingw-w64 for Windows cross-compilation..."
        
        # Detect package manager and install
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
            exit 1
        fi
    fi
}

# Check Windows cross-compilation tools
check_windows_tools

echo
echo "[3/6] Downloading dependencies..."
go mod download

echo
echo "[4/6] Building for multiple platforms..."

# Create build directories
mkdir -p build/windows
mkdir -p build/linux
mkdir -p build/darwin

# Build for Windows (amd64)
echo "Building for Windows (amd64)..."
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc
go build -ldflags="-s -w" -o build/windows/whatsapp-h2h.exe cmd/server/main.go

# Build for Linux (amd64)
echo "Building for Linux (amd64)..."
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=1
export CC=gcc
go build -ldflags="-s -w" -o build/linux/whatsapp-h2h cmd/server/main.go

# Build for macOS (amd64)
echo "Building for macOS (amd64)..."
export GOOS=darwin
export GOARCH=amd64
export CGO_ENABLED=1
export CC=clang
go build -ldflags="-s -w" -o build/darwin/whatsapp-h2h cmd/server/main.go

# Build for macOS (arm64) - Apple Silicon
echo "Building for macOS (arm64)..."
export GOOS=darwin
export GOARCH=arm64
export CGO_ENABLED=1
export CC=clang
go build -ldflags="-s -w" -o build/darwin/whatsapp-h2h-arm64 cmd/server/main.go

echo
echo "[5/6] Creating deployment packages..."

# Function to create deployment package
create_deployment_package() {
    local platform=$1
    local dir="build/$platform"
    
    echo "Creating deployment package for $platform..."
    
    # Copy necessary files
    cp .env.example "$dir/" 2>/dev/null || true
    cp README.md "$dir/" 2>/dev/null || true
    cp MIGRATION_NOTES.md "$dir/" 2>/dev/null || true
    
    # Create directories
    mkdir -p "$dir/db"
    mkdir -p "$dir/logs"
    
    if [ "$platform" = "windows" ]; then
        # Windows batch files
        cat > "$dir/start.bat" << 'EOF'
@echo off
echo Starting WhatsApp H2H Otomax...
whatsapp-h2h.exe
pause
EOF

        cat > "$dir/install-service.bat" << 'EOF'
@echo off
echo Installing as Windows Service...
sc create "WhatsAppH2H" binPath= "%~dp0whatsapp-h2h.exe" start= auto
sc description "WhatsAppH2H" "WhatsApp Host-to-Host middleware for Otomax"
echo Service installed. Use 'sc start WhatsAppH2H' to start.
pause
EOF

        cat > "$dir/uninstall-service.bat" << 'EOF'
@echo off
echo Uninstalling Windows Service...
sc stop "WhatsAppH2H"
sc delete "WhatsAppH2H"
echo Service uninstalled.
pause
EOF

        # PowerShell script
        cat > "$dir/start.ps1" << 'EOF'
# WhatsApp H2H Otomax - PowerShell Launcher
Write-Host "Starting WhatsApp H2H Otomax..." -ForegroundColor Green
Write-Host "Press Ctrl+C to stop the application" -ForegroundColor Yellow
Write-Host ""
& ".\whatsapp-h2h.exe"
EOF

    else
        # Unix shell scripts
        cat > "$dir/start.sh" << 'EOF'
#!/bin/bash
echo "Starting WhatsApp H2H Otomax..."
echo "Press Ctrl+C to stop the application"
echo ""
./whatsapp-h2h
EOF
        chmod +x "$dir/start.sh"
        
        # Systemd service file for Linux
        if [ "$platform" = "linux" ]; then
            cat > "$dir/whatsapp-h2h.service" << 'EOF'
[Unit]
Description=WhatsApp H2H Otomax Service
After=network.target

[Service]
Type=simple
User=whatsapp
WorkingDirectory=/opt/whatsapp-h2h
ExecStart=/opt/whatsapp-h2h/whatsapp-h2h
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
        fi
    fi
}

# Create deployment packages for each platform
create_deployment_package "windows"
create_deployment_package "linux"
create_deployment_package "darwin"

echo
echo "[6/6] Creating installation guide..."

# Create comprehensive installation guide
cat > build/INSTALLATION.md << 'EOF'
# WhatsApp H2H Otomax - Installation Guide

## Windows Installation

### Method 1: Direct Execution
1. Copy the `windows/` folder to your Windows machine
2. Edit `windows/.env` with your configuration
3. Double-click `start.bat` to run the application

### Method 2: Windows Service
1. Copy the `windows/` folder to your Windows machine
2. Edit `windows/.env` with your configuration
3. Run `install-service.bat` as Administrator
4. Start the service: `sc start WhatsAppH2H`
5. To uninstall: Run `uninstall-service.bat` as Administrator

### Method 3: PowerShell
1. Copy the `windows/` folder to your Windows machine
2. Edit `windows/.env` with your configuration
3. Run PowerShell as Administrator
4. Execute: `.\start.ps1`

## Linux Installation

### Method 1: Direct Execution
1. Copy the `linux/` folder to your Linux machine
2. Edit `linux/.env` with your configuration
3. Run: `chmod +x whatsapp-h2h && ./start.sh`

### Method 2: Systemd Service
1. Copy the `linux/` folder to `/opt/whatsapp-h2h/`
2. Create user: `sudo useradd -r -s /bin/false whatsapp`
3. Set ownership: `sudo chown -R whatsapp:whatsapp /opt/whatsapp-h2h/`
4. Copy service file: `sudo cp whatsapp-h2h.service /etc/systemd/system/`
5. Enable and start: `sudo systemctl enable whatsapp-h2h && sudo systemctl start whatsapp-h2h`

## macOS Installation

### Method 1: Direct Execution
1. Copy the `darwin/` folder to your macOS machine
2. Edit `darwin/.env` with your configuration
3. Run: `chmod +x whatsapp-h2h && ./start.sh`

### Method 2: LaunchAgent (Background Service)
1. Copy the `darwin/` folder to `/Applications/WhatsAppH2H/`
2. Create LaunchAgent plist file
3. Load the service: `launchctl load ~/Library/LaunchAgents/com.whatsapp.h2h.plist`

## Configuration

Edit the `.env` file in your chosen platform directory:

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

1. Start the application
2. Check for `whatsapp-qrcode.png` file
3. Scan the QR code with WhatsApp: Settings > Linked Devices > Link a Device
4. The application will connect and start the HTTP server
5. Test with: `curl http://localhost:8080/health`

## Troubleshooting

### Windows
- If antivirus blocks the executable, add it to exclusions
- For service installation, run as Administrator
- Check Windows Firewall settings

### Linux
- Ensure the executable has execute permissions
- Check systemd logs: `journalctl -u whatsapp-h2h -f`
- Verify port 8080 is not in use: `netstat -tlnp | grep 8080`

### macOS
- Allow the application in Security & Privacy settings
- Check Console.app for application logs
- Verify port 8080 is not in use: `lsof -i :8080`
EOF

echo
echo "========================================"
echo "CROSS-PLATFORM BUILD COMPLETED!"
echo "========================================"
echo
echo "Build outputs:"
echo "├── build/windows/     - Windows executable and scripts"
echo "├── build/linux/       - Linux executable and scripts"
echo "├── build/darwin/      - macOS executables (amd64 + arm64)"
echo "└── build/INSTALLATION.md - Installation guide"
echo
echo "File sizes:"
echo "Windows:"
ls -lh build/windows/whatsapp-h2h.exe 2>/dev/null || echo "  whatsapp-h2h.exe: N/A"
echo "Linux:"
ls -lh build/linux/whatsapp-h2h 2>/dev/null || echo "  whatsapp-h2h: N/A"
echo "macOS:"
ls -lh build/darwin/whatsapp-h2h* 2>/dev/null || echo "  whatsapp-h2h: N/A"
echo
echo "Next steps:"
echo "1. Copy the appropriate platform folder to your target machine"
echo "2. Edit the .env file with your configuration"
echo "3. Follow the installation guide in build/INSTALLATION.md"
echo "4. Start the application and scan the QR code"

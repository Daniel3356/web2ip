#!/bin/bash

# Web2IP High-Performance Scanner Installation Script
# For Raspberry Pi 5 with 8GB RAM

set -e

echo "🚀 Web2IP High-Performance Scanner Installation"
echo "==============================================="

# Check if running on Pi 5
if [ "$(uname -m)" != "aarch64" ] && [ "$(uname -m)" != "armv7l" ]; then
    echo "⚠️  Warning: This script is optimized for Raspberry Pi 5"
    echo "   It may work on other systems but performance may vary"
fi

# Check available memory
TOTAL_MEM=$(free -m | awk 'NR==2{print $2}')
if [ "$TOTAL_MEM" -lt 7000 ]; then
    echo "⚠️  Warning: System has less than 8GB RAM ($TOTAL_MEM MB)"
    echo "   High performance mode may not perform optimally"
fi

# Create necessary directories
echo "📁 Creating directories..."
sudo mkdir -p /var/log/web2ip
sudo chown pi:pi /var/log/web2ip

# Build the scanner
echo "🔨 Building Web2IP scanner..."
go build -o web2ip

# Make executable
chmod +x web2ip

# Install systemd service
echo "🔧 Installing systemd service..."
sudo cp systemd/web2ip-scanner.service /etc/systemd/system/
sudo systemctl daemon-reload

# Create CSV file if it doesn't exist
if [ ! -f "top10milliondomains.csv" ]; then
    echo "📄 Creating sample CSV file..."
    echo -e "rank,domain\n1,google.com\n2,youtube.com\n3,facebook.com\n4,twitter.com\n5,instagram.com\n6,amazon.com\n7,wikipedia.org\n8,github.com\n9,stackoverflow.com\n10,reddit.com" > top10milliondomains.csv
fi

echo "✅ Installation complete!"
echo ""
echo "Usage:"
echo "  ./web2ip                    # Standard mode"
echo "  ./web2ip -high-performance  # High performance mode (800 workers)"
echo "  ./web2ip -help             # Show help"
echo ""
echo "24/7 Operation:"
echo "  sudo systemctl enable web2ip-scanner"
echo "  sudo systemctl start web2ip-scanner"
echo "  sudo systemctl status web2ip-scanner"
echo ""
echo "Monitoring:"
echo "  sudo journalctl -u web2ip-scanner -f"
echo "  tail -f /var/log/web2ip/scanner.log"
echo ""
echo "🔥 Ready for high-performance scanning!"
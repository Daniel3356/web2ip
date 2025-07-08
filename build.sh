#!/bin/bash

echo "🔨 Building Recon Scanner for Raspberry Pi 5..."

# Check if we're on ARM64
ARCH=$(uname -m)
if [[ "$ARCH" == "aarch64" ]]; then
    echo "✅ Detected ARM64 architecture (Raspberry Pi)"
else
    echo "⚠️ Warning: Not running on ARM64, build may not be optimized"
fi

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -f recon-scanner

# Set Go environment variables for ARM64 optimization
export GOOS=linux
export GOARCH=arm64
export CGO_ENABLED=1

# Install dependencies
echo "📦 Installing dependencies..."
go mod tidy

# Build with optimizations for ARM64
echo "🔧 Compiling application with ARM64 optimizations..."
go build -ldflags="-s -w" -tags netgo -o recon-scanner main.go

# Check if build was successful
if [ -f "./recon-scanner" ]; then
    echo "✅ Build complete! Executable created successfully."
    echo "📋 File info:"
    ls -la recon-scanner
    
    # Make executable
    chmod +x recon-scanner
    
    echo ""
    echo "🚀 Ready to run:"
    echo "   ./recon-scanner"
    echo ""
    echo "📊 Performance modes:"
    echo "   🌙 Full Power: 1:37 AM - 6:30 AM (Toronto time)"
    echo "   ☀️ Conservation: 6:30 AM - 1:37 AM"
    echo ""
    echo "💡 The scanner will automatically adjust performance based on time of day"
    echo "💡 Check logs with: tail -f recon.log"
else
    echo "❌ Build failed! No executable was created."
    exit 1
fi
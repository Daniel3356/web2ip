#!/bin/bash

echo "Building Recon Scanner for macOS testing..."

# Clean previous builds
echo "Cleaning previous builds..."
rm -f recon-scanner

# Build for macOS (no cross-compilation issues)
echo "Compiling for macOS..."
go build -o recon-scanner main.go

# Check if build was successful
if [ -f "./recon-scanner" ]; then
    echo "Build complete! Executable created successfully."
    chmod +x recon-scanner
    echo "Ready to test on macOS with: ./recon-scanner"
else
    echo "Build failed!"
    exit 1
fi
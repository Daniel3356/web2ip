#!/bin/bash

echo "🔨 Building Recon Scanner..."

# Install dependencies
go mod tidy

# Build the application
go build -o recon-scanner main.go

if [ $? -eq 0 ]; then
    echo "✅ Build complete! Run with: ./recon-scanner"
else
    echo "❌ Build failed!"
    exit 1
fi
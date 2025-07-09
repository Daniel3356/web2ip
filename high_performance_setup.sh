#!/bin/bash

# High-Performance Web2IP Configuration and Monitoring Script
# Optimized for Raspberry Pi 5 with 8GB RAM

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="recon-scanner"
LOG_FILE="recon.log"
DB_FILE="recon_results.db"
CSV_FILE="top10milliondomains.csv"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${CYAN}$1${NC}"
}

# Function to check system requirements
check_system_requirements() {
    print_header "🔍 Checking System Requirements..."
    
    # Check if we're on Raspberry Pi
    if [ -f "/sys/firmware/devicetree/base/model" ]; then
        MODEL=$(cat /sys/firmware/devicetree/base/model)
        print_status "Detected: $MODEL"
    else
        print_warning "Not running on Raspberry Pi - thermal monitoring may not work"
    fi
    
    # Check RAM
    TOTAL_RAM=$(free -m | awk 'NR==2{print $2}')
    if [ "$TOTAL_RAM" -lt 7000 ]; then
        print_warning "Only ${TOTAL_RAM}MB RAM detected - recommended 8GB for high-performance mode"
    else
        print_status "RAM: ${TOTAL_RAM}MB - OK for high-performance mode"
    fi
    
    # Check CPU cores
    CPU_CORES=$(nproc)
    print_status "CPU Cores: $CPU_CORES"
    
    # Check CPU temperature
    if [ -f "/sys/class/thermal/thermal_zone0/temp" ]; then
        TEMP_RAW=$(cat /sys/class/thermal/thermal_zone0/temp)
        TEMP_C=$((TEMP_RAW / 1000))
        if [ "$TEMP_C" -gt 60 ]; then
            print_warning "CPU Temperature: ${TEMP_C}°C - High temperature detected"
        else
            print_status "CPU Temperature: ${TEMP_C}°C - OK"
        fi
    else
        print_warning "Cannot read CPU temperature - thermal monitoring may not work"
    fi
    
    # Check disk space
    DISK_USAGE=$(df -h . | awk 'NR==2{print $5}' | sed 's/%//')
    if [ "$DISK_USAGE" -gt 80 ]; then
        print_warning "Disk usage: ${DISK_USAGE}% - Consider cleaning up"
    else
        print_status "Disk usage: ${DISK_USAGE}% - OK"
    fi
    
    # Check for SSD (recommended for database)
    if mount | grep -q "on / type ext4"; then
        print_status "Root filesystem: ext4 - OK"
    fi
    
    echo
}

# Function to check cooling system
check_cooling() {
    print_header "🌡️ Checking Cooling System..."
    
    # Check if we can read thermal zones
    if [ -d "/sys/class/thermal" ]; then
        for zone in /sys/class/thermal/thermal_zone*; do
            if [ -f "$zone/type" ] && [ -f "$zone/temp" ]; then
                ZONE_TYPE=$(cat "$zone/type")
                ZONE_TEMP=$(cat "$zone/temp")
                ZONE_TEMP_C=$((ZONE_TEMP / 1000))
                print_status "Thermal Zone: $ZONE_TYPE = ${ZONE_TEMP_C}°C"
            fi
        done
    else
        print_warning "Cannot access thermal zones"
    fi
    
    # Check for active cooling
    if [ -d "/sys/class/hwmon" ]; then
        for hwmon in /sys/class/hwmon/hwmon*; do
            if [ -f "$hwmon/name" ]; then
                HWMON_NAME=$(cat "$hwmon/name")
                print_status "Hardware Monitor: $HWMON_NAME"
            fi
        done
    fi
    
    echo
}

# Function to optimize system settings
optimize_system() {
    print_header "⚡ Optimizing System Settings..."
    
    # Increase file descriptor limits
    print_status "Setting file descriptor limits..."
    echo "* soft nofile 65536" | sudo tee -a /etc/security/limits.conf > /dev/null
    echo "* hard nofile 65536" | sudo tee -a /etc/security/limits.conf > /dev/null
    
    # Optimize network settings
    print_status "Optimizing network settings..."
    echo "net.core.somaxconn = 65536" | sudo tee -a /etc/sysctl.conf > /dev/null
    echo "net.ipv4.tcp_max_syn_backlog = 65536" | sudo tee -a /etc/sysctl.conf > /dev/null
    echo "net.core.netdev_max_backlog = 30000" | sudo tee -a /etc/sysctl.conf > /dev/null
    
    # Apply sysctl changes
    sudo sysctl -p > /dev/null
    
    # Optimize Go runtime
    export GOMAXPROCS=$(nproc)
    export GOGC=100
    
    print_status "System optimization complete"
    echo
}

# Function to build the application
build_application() {
    print_header "🔨 Building Application..."
    
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found - are you in the correct directory?"
        exit 1
    fi
    
    print_status "Building recon-scanner..."
    go build -o "$BINARY_NAME" -ldflags="-s -w" .
    
    if [ -f "$BINARY_NAME" ]; then
        print_status "Build successful - binary: $BINARY_NAME"
    else
        print_error "Build failed"
        exit 1
    fi
    
    echo
}

# Function to run performance test
run_performance_test() {
    print_header "🚀 Running Performance Test..."
    
    if [ ! -f "test_high_performance.go" ]; then
        print_error "test_high_performance.go not found"
        exit 1
    fi
    
    print_status "Running high-performance test..."
    timeout 60 go run test_high_performance.go || {
        print_warning "Test timed out or was interrupted"
    }
    
    echo
}

# Function to start high-performance mode
start_high_performance() {
    print_header "🚀 Starting High-Performance Mode..."
    
    if [ ! -f "$BINARY_NAME" ]; then
        print_error "Binary not found - run build first"
        exit 1
    fi
    
    if [ ! -f "$CSV_FILE" ]; then
        print_warning "CSV file not found - creating sample file"
        echo "rank,domain" > "$CSV_FILE"
        echo "1,google.com" >> "$CSV_FILE"
        echo "2,github.com" >> "$CSV_FILE"
        echo "3,stackoverflow.com" >> "$CSV_FILE"
    fi
    
    print_status "Starting scanner in high-performance mode..."
    print_status "Log file: $LOG_FILE"
    print_status "Database: $DB_FILE"
    print_status "CSV file: $CSV_FILE"
    
    # Start with detailed logging
    ./"$BINARY_NAME" --high-performance --detailed-logging
}

# Function to monitor system resources
monitor_resources() {
    print_header "📊 System Resource Monitor..."
    
    while true; do
        clear
        echo -e "${CYAN}=== SYSTEM RESOURCE MONITOR ===${NC}"
        echo -e "${CYAN}Press Ctrl+C to exit${NC}"
        echo
        
        # CPU Temperature
        if [ -f "/sys/class/thermal/thermal_zone0/temp" ]; then
            TEMP_RAW=$(cat /sys/class/thermal/thermal_zone0/temp)
            TEMP_C=$((TEMP_RAW / 1000))
            if [ "$TEMP_C" -gt 70 ]; then
                echo -e "🌡️  CPU Temperature: ${RED}${TEMP_C}°C${NC}"
            elif [ "$TEMP_C" -gt 60 ]; then
                echo -e "🌡️  CPU Temperature: ${YELLOW}${TEMP_C}°C${NC}"
            else
                echo -e "🌡️  CPU Temperature: ${GREEN}${TEMP_C}°C${NC}"
            fi
        fi
        
        # Memory Usage
        MEM_INFO=$(free -m | awk 'NR==2{printf "%.1f%%", $3*100/$2}')
        MEM_USED=$(free -m | awk 'NR==2{print $3}')
        MEM_TOTAL=$(free -m | awk 'NR==2{print $2}')
        echo -e "🧠 Memory Usage: ${GREEN}${MEM_INFO}${NC} (${MEM_USED}/${MEM_TOTAL}MB)"
        
        # CPU Load
        CPU_LOAD=$(uptime | awk -F'load average:' '{print $2}' | sed 's/,//g')
        echo -e "🔄 CPU Load:${GREEN}${CPU_LOAD}${NC}"
        
        # Disk Usage
        DISK_INFO=$(df -h . | awk 'NR==2{print $5}')
        echo -e "💾 Disk Usage: ${GREEN}${DISK_INFO}${NC}"
        
        # Network connections
        if command -v ss > /dev/null; then
            CONN_COUNT=$(ss -tan | grep -c ESTAB)
            echo -e "🌐 Network Connections: ${GREEN}${CONN_COUNT}${NC}"
        fi
        
        # Process info
        if pgrep -f "$BINARY_NAME" > /dev/null; then
            PID=$(pgrep -f "$BINARY_NAME")
            CPU_USAGE=$(ps -p "$PID" -o %cpu --no-headers | sed 's/ //')
            MEM_USAGE=$(ps -p "$PID" -o %mem --no-headers | sed 's/ //')
            echo -e "🎯 Scanner Process: ${GREEN}Running${NC} (PID: $PID, CPU: ${CPU_USAGE}%, MEM: ${MEM_USAGE}%)"
        else
            echo -e "🎯 Scanner Process: ${RED}Not Running${NC}"
        fi
        
        echo
        echo -e "${CYAN}=== LOG TAIL ===${NC}"
        if [ -f "$LOG_FILE" ]; then
            tail -5 "$LOG_FILE"
        else
            echo "No log file found"
        fi
        
        sleep 5
    done
}

# Function to show usage
show_usage() {
    echo -e "${CYAN}High-Performance Web2IP Configuration Script${NC}"
    echo
    echo "Usage: $0 [command]"
    echo
    echo "Commands:"
    echo "  check         - Check system requirements and cooling"
    echo "  optimize      - Optimize system settings for high-performance"
    echo "  build         - Build the application"
    echo "  test          - Run performance test"
    echo "  start         - Start high-performance mode"
    echo "  monitor       - Monitor system resources"
    echo "  help          - Show this help message"
    echo
    echo "Examples:"
    echo "  $0 check      # Check system requirements"
    echo "  $0 optimize   # Optimize system settings"
    echo "  $0 build      # Build application"
    echo "  $0 test       # Run performance test"
    echo "  $0 start      # Start high-performance scanning"
    echo "  $0 monitor    # Monitor system resources"
    echo
}

# Main script
case "${1:-help}" in
    "check")
        check_system_requirements
        check_cooling
        ;;
    "optimize")
        optimize_system
        ;;
    "build")
        build_application
        ;;
    "test")
        run_performance_test
        ;;
    "start")
        start_high_performance
        ;;
    "monitor")
        monitor_resources
        ;;
    "help"|*)
        show_usage
        ;;
esac
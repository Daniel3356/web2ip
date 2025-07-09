#!/bin/bash

echo "Starting High-Performance Web2IP Scanner for Raspberry Pi 5"
echo "Optimized for 800 workers and 8GB RAM"

# Set system limits
ulimit -n 65536
ulimit -u 32768

# Set CPU governor to performance
echo "Setting CPU governor to performance mode"
sudo cpufreq-set -g performance

# Increase network buffers
echo "Optimizing network buffers"
sudo sysctl -w net.core.rmem_max=16777216
sudo sysctl -w net.core.wmem_max=16777216
sudo sysctl -w net.core.netdev_max_backlog=5000

# Set process priority
echo "Setting high priority for scanner process"
sudo nice -n -10 ./main_high_performance &

PID=$!
echo "High-performance scanner started with PID: $PID"

# Monitor the process
while kill -0 $PID 2>/dev/null; do
    sleep 60
    echo "Scanner running - PID: $PID, Time: $(date)"
done

echo "Scanner process finished"
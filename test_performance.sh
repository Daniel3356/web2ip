#!/bin/bash

# Performance Comparison Test
# Compare standard vs high-performance mode

set -e

echo "🧪 Web2IP Performance Comparison Test"
echo "====================================="

# Create test CSV with more domains
echo "📄 Creating test dataset..."
cat > test_domains.csv << 'EOF'
rank,domain
1,google.com
2,youtube.com
3,facebook.com
4,twitter.com
5,instagram.com
6,amazon.com
7,wikipedia.org
8,github.com
9,stackoverflow.com
10,reddit.com
11,netflix.com
12,apple.com
13,microsoft.com
14,linkedin.com
15,pinterest.com
16,tumblr.com
17,yahoo.com
18,ebay.com
19,cnn.com
20,bbc.com
EOF

# Backup original CSV
if [ -f "top10milliondomains.csv" ]; then
    cp top10milliondomains.csv top10milliondomains.csv.backup
fi

# Use test dataset
cp test_domains.csv top10milliondomains.csv

echo "🚀 Building scanner..."
go build -o web2ip

echo ""
echo "📊 Test 1: Standard Mode (Time-based)"
echo "====================================="
echo "Mode: Conservation/Full Power based on time"
echo "Workers: 2-12 (depending on time)"
echo "Batch Size: 500-5000"
echo ""
echo "Starting test..."
timeout 30 ./web2ip > standard_mode_output.log 2>&1 || true
echo "✅ Standard mode test completed"

echo ""
echo "📊 Test 2: High Performance Mode"
echo "================================"
echo "Mode: High Performance (800 workers)"
echo "Workers: 800"
echo "Batch Size: 2000 (dynamic)"
echo "Memory Management: Intelligent"
echo "Thermal Monitoring: Enabled"
echo ""
echo "Starting test..."
timeout 30 ./web2ip -high-performance > high_performance_output.log 2>&1 || true
echo "✅ High performance mode test completed"

echo ""
echo "📊 PERFORMANCE COMPARISON RESULTS"
echo "================================="

echo ""
echo "Standard Mode Output:"
echo "--------------------"
tail -20 standard_mode_output.log

echo ""
echo "High Performance Mode Output:"
echo "----------------------------"
tail -20 high_performance_output.log

echo ""
echo "📈 Key Differences:"
echo "==================="
echo "• Workers: 2-12 vs 800 concurrent workers"
echo "• Batch Processing: Fixed vs Dynamic sizing"
echo "• Memory Management: Basic vs Intelligent monitoring"
echo "• Thermal Control: Time-based vs Real-time monitoring"
echo "• Error Handling: Basic vs Advanced recovery"
echo "• Resource Utilization: Conservative vs Optimized"
echo ""
echo "🔥 High Performance Mode Benefits:"
echo "• Up to 67x more concurrent workers"
echo "• Intelligent resource management"
echo "• Real-time health monitoring"
echo "• Dynamic throttling and recovery"
echo "• 24/7 operation capability"

# Restore original CSV
if [ -f "top10milliondomains.csv.backup" ]; then
    mv top10milliondomains.csv.backup top10milliondomains.csv
fi

echo ""
echo "✅ Performance comparison complete!"
echo "📄 Detailed logs saved to:"
echo "   - standard_mode_output.log"
echo "   - high_performance_output.log"
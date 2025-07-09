#!/bin/bash

CSV="top10milliondomains.csv"
DB="recon_results.db"

# Get total domains (strip header if it exists)
TOTAL=$(wc -l < "$CSV")
if head -1 "$CSV" | grep -iq "domain"; then
    TOTAL=$((TOTAL - 1))
fi

while true; do
    # Get processed domains from the DB
    DONE=$(sqlite3 "$DB" "SELECT COUNT(*) FROM domains;")
    [ -z "$DONE" ] && DONE=0

    # Calculate percent
    if [ "$TOTAL" -gt 0 ]; then
        PERCENT=$(awk "BEGIN { printf \"%.2f\", ($DONE/$TOTAL)*100 }")
    else
        PERCENT="?"
    fi

    # Estimate ETA in days
    # Read the first and latest processed_at for timing
    START_TIME=$(sqlite3 "$DB" "SELECT MIN(processed_at) FROM domains WHERE processed_at IS NOT NULL;")
    LAST_TIME=$(sqlite3 "$DB" "SELECT MAX(processed_at) FROM domains WHERE processed_at IS NOT NULL;")

    if [ -n "$START_TIME" ] && [ -n "$LAST_TIME" ] && [ "$DONE" -gt 10 ]; then
        # Convert to seconds since epoch
        START_SEC=$(date -d "$START_TIME" +%s 2>/dev/null || gdate -d "$START_TIME" +%s)
        END_SEC=$(date -d "$LAST_TIME" +%s 2>/dev/null || gdate -d "$LAST_TIME" +%s)
        ELAPSED=$((END_SEC - START_SEC))
        RATE=$(awk "BEGIN { if($ELAPSED>0) print $DONE/$ELAPSED; else print 0 }")
        REMAIN=$((TOTAL - DONE))
        if (( $(echo "$RATE > 0" | bc -l) )); then
            ETA_SEC=$(awk "BEGIN { print $REMAIN/$RATE }")
            ETA_DAYS=$(awk "BEGIN { printf \"%.2f\", $ETA_SEC/86400 }")
        else
            ETA_DAYS="?"
        fi
    else
        ETA_DAYS="estimating"
    fi

    echo "Progress: $DONE / $TOTAL ($PERCENT%)"
    echo "Estimated time remaining: $ETA_DAYS days"
    echo "Last update: $(date)"
    echo "------------------------------"
    sleep 30
done

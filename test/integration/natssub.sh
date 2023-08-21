#!/bin/bash

INSTANCES_DB="$HOME/.cedana/instances.db"

JOB_ID=$(sqlite3 "$INSTANCES_DB" "SELECT cedana_id FROM instances where tag='worker' LIMIT 1;") && \
WORKER_ID=$(sqlite3 "$INSTANCES_DB" "SELECT allocated_id FROM instances where tag='worker' LIMIT 1;") && \

# Define channels to subscribe to
CHAN="CEDANA.${JOB_ID}.${WORKER_ID}.commands"

LOG_FILE="messages.log"

# Start subscribing to the NATS channel and log messages
nats sub "$CHAN" > "$LOG_FILE" &
NATS_SUB_PID=$!

# Sleep for 5 seconds
sleep 20

# Stop the NATS subscription
kill "$NATS_SUB_PID" 2>/dev/null

LOG_FILE="messages.log"
PATTERN="Received on \"$CHAN\""

# Count the matched lines in the log file
COUNT=$(grep -c "$PATTERN" "$LOG_FILE")

echo "Total messages with pattern: $COUNT"


echo "Subscription stopped."

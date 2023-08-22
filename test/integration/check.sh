#!/usr/bin/env bash

# Your code to generate the log file goes here
# ...

# Perform the check using grep
if grep -q '"checkpoint_state":"CHECKPOINT_FAILED"' /tmp/messages.log; then
    echo "Test failed: CHECKPOINT_FAILED found"
    exit 1
else
    echo "Test passed: No CHECKPOINT_FAILED found"
    exit 0
fi

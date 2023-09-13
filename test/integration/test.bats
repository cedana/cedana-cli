#!/usr/bin/env bats

# Have if env vars are not set, then use a config file maybe
INIT_CEDANA_CLI="$CEDANA_CLI_PATH"
YMLDIR="$BATS_TEST_DIRNAME/jobs"
YML="job.yml"
INSTANCES_DB="$HOME/.cedana/instances.db"
LOG_FILE="$HOME/tmp/messages.log"



@test "Checking if aws cli is installed" {
    run aws --version
    [ "$status" -eq 0 ]
}

@test "Checking if cedana-cli executable exists" {
    skip
    [[ -x "$INIT_CEDANA_CLI" ]]
}

@test "checking if job.yml exists in $YMLDIR" {
    [[ -e "$YMLDIR/$YML" ]]
}

@test "Check expected heartbeat_enabled value" {
  run jq '.checkpoint.heartbeat_enabled' $HOME/.cedana/cedana_config.json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "true" ]
  echo "Make sure heartbeat_enabled is set to true and ./$INIT_CEDANA_CLI bootstrap has been run"
}

@test "Check expected keep_running value" {
  run jq '.keep_running' $HOME/.cedana/cedana_config.json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "true" ]
}

@test "Check expected heartbeat_interval_seconds value" {
  run jq '.checkpoint.heartbeat_interval_seconds' $HOME/.cedana/cedana_config.json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "0" ]
}

@test "Run job on instance" {
  skip
  run sudo ./build/cedana-cli run test/integration/jobs/job.yml > $HOME/tmp/log_output.txt
  # Test passed if success signal is received
  [ "$status" -eq 0 ]
}


@test "Checking if instances.db exists" {
  skip
    [[ -f "$INSTANCES_DB" ]]
}


@test "Check # of messages received on channel" {
  skip
  JOB_ID=$(sqlite3 "$INSTANCES_DB" "SELECT job_id FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID_JSON=$(sqlite3 "$INSTANCES_DB" "SELECT instances FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID=$(echo "$WORKER_ID_JSON" | jq -r '.[0].instance_id')

  # Define channels to subscribe to
  CHAN="CEDANA.${JOB_ID}.${WORKER_ID}.commands"

  LOG_FILE="$HOME/tmp/messages.log"

  # Start subscribing to the NATS channel and log messages
  nats sub "$CHAN" > "$LOG_FILE" &
  NATS_SUB_PID=$!
  # Sleep for 5 seconds
  sleep 20
  # Stop the NATS subscription
  kill "$NATS_SUB_PID" 2>/dev/null
  PATTERN="Received on \"$CHAN\""

  # Count the matched lines in the log file
  COUNT=$(grep -c "$PATTERN" "$LOG_FILE")
  echo $COUNT
  [[ "$COUNT" -eq 3 ]]
}

@test "Testing check point messages" {
  skip
  # skip
  JOB_ID=$(sqlite3 "$INSTANCES_DB" "SELECT job_id FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID_JSON=$(sqlite3 "$INSTANCES_DB" "SELECT instances FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID=$(echo "$WORKER_ID_JSON" | jq -r '.[0].instance_id')

  # Define channels to subscribe to
  CHAN="CEDANA.${JOB_ID}.${WORKER_ID}.commands"

  LOG_FILE="$HOME/tmp/messages.log"

  # Start subscribing to the NATS channel and log messages
  nats sub "$CHAN" > "$LOG_FILE" &
  NATS_SUB_PID=$!

  run sudo ./build/cedana-cli whisper checkpoint -j $JOB_ID

  [[ "$status" -eq 0 ]]

  sleep 20
  # Stop the NATS subscription
  kill "$NATS_SUB_PID" 2>/dev/null
  PATTERN="Received on \"$CHAN\""

  # Count the matched lines in the log file
  COUNT=$(grep -c "$PATTERN" "$LOG_FILE")
  echo $COUNT


  # Test passed if success signal is received
  echo Job ID: $JOB_ID


  [[ "$COUNT" -eq 3 || "$COUNT" -eq 4 || "$COUNT" -eq 5 || "$COUNT" -eq 6 ]]

}

@test "Check for failed checkpoint" {
  skip
  ./test/integration/check.sh
  [[ "$status" -eq 0 ]]
}

@test "Testing whisper restore" {
  skip
  JOB_ID=$(sqlite3 "$INSTANCES_DB" "SELECT job_id FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID_JSON=$(sqlite3 "$INSTANCES_DB" "SELECT instances FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  WORKER_ID=$(echo "$WORKER_ID_JSON" | jq -r '.[0].instance_id')

  # Define channels to subscribe to
  CHAN="CEDANA.${JOB_ID}.${WORKER_ID}.state"

  LOG_FILE="$HOME/tmp/messages.log"
  # Start subscribing to the NATS channel and log messages
  nats sub "$CHAN" > "$LOG_FILE" &
  NATS_SUB_PID=$!

  JOB_ID=$(sqlite3 "$INSTANCES_DB" "SELECT job_id FROM jobs ORDER BY created_at DESC LIMIT 1;") && \
  run sudo ./build/cedana-cli whisper restore -j $JOB_ID -l

  sleep 5
  # Stop the NATS subscription
  kill "$NATS_SUB_PID" 2>/dev/null
  PATTERN="Received on \"$CHAN\""

  # Count the matched lines in the log file
  # Skipping this test for now, there is an issue with the COUNT for some reason after echo it is literally equal to $PATTERN...
  COUNT=$(grep -c "$PATTERN" "$HOME/tmp/messages.log")
  echo $COUNT
  echo $LOG_FILE

  [[ "$COUNT" -eq 1 ]]
  [[ "$status" -eq 0 ]]
}
# Add check for successful restore msg

# Add check for failed restore msg

@test "Check # of running instances - before destroy" {
  skip
  # Get the instance IDs of running instances
  instance_ids=$(aws ec2 describe-instances --filters "Name=instance-state-name,Values=running" --query "Reservations[].Instances[].InstanceId" --output text)

  # Count the number of instance IDs
  num_instances=$(echo "$instance_ids" | wc -w)

  echo "Number of running EC2 instances: $num_instances"

  # Test passed if number of running instances is 1
  [ "$num_instances" -eq 1 ]

}

@test "Tear down all instances" {
  skip
  run sudo ./build/cedana-cli destroy-all > $HOME/tmp/log_output.txt

  # Test passed if success signal is received
  [ "$status" -eq 0 ]
}

@test "Check # of running instances - after destroy" {
  skip
  # Get the instance IDs of running instances
  instance_ids=$(aws ec2 describe-instances --filters "Name=instance-state-name,Values=running" --query "Reservations[].Instances[].InstanceId" --output text)

  # Count the number of instance IDs
  num_instances=$(echo "$instance_ids" | wc -w)

  echo "Number of running EC2 instances: $num_instances"

  # Test passed if number of running instances is 1
  [ "$num_instances" -eq 0 ]

}

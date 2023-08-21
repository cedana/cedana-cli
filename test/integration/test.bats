#!/usr/bin/env bats

INIT_CEDANA_CLI="$BATS_TEST_DIRNAME/../../cedana-cli"
YMLDIR="$BATS_TEST_DIRNAME/jobs"
YML="job.yml"

@test "Checking if aws cli is installed" {
    run aws --version
    [ "$status" -eq 0 ]
}

@test "Checking if cedana-cli executable exists" {
    [[ -x "$INIT_CEDANA_CLI" ]]
}

@test "checking if job.yml exists in $YMLDIR" {
    [[ -e "$YMLDIR/$YML" ]]
}

@test "Check expected heartbeat_enabled value" {
  run jq '.checkpoint.heartbeat_enabled' $HOME/.cedana/cedana_config.json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "true" ]
  echo "Make sure heartbeat_enabled is set to true and ./cedana-cli bootstrap has been run"
}

@test "Check expected keep_running value" {
  run jq '.keep_running' $HOME/.cedana/cedana_config.json
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "true" ]
}

@test "Run job on instance" {
  # skip
    run ./cedana-cli run $YML > $BATS_TMPDIR/log_output.txt

    # Test passed if success signal is received
    [ "$status" -eq 0 ]

    echo $BATS_TMPDIR/log_output.txt
}

@test "Check # of running instances - before destroy" {
  # skip
  # Get the instance IDs of running instances
  instance_ids=$(aws ec2 describe-instances --filters "Name=instance-state-name,Values=running" --query "Reservations[].Instances[].InstanceId" --output text)

  # Count the number of instance IDs
  num_instances=$(echo "$instance_ids" | wc -w)

  echo "Number of running EC2 instances: $num_instances"

  # Test passed if number of running instances is 1
  [ "$num_instances" -eq 1 ]

}

@test "Tear down all instances" {
  run ./cedana-cli destroy-all > $BATS_TMPDIR/log_output.txt

  # Test passed if success signal is received
  [ "$status" -eq 0 ]
}

@test "Check # of running instances - after destroy" {
  # Get the instance IDs of running instances
  instance_ids=$(aws ec2 describe-instances --filters "Name=instance-state-name,Values=running" --query "Reservations[].Instances[].InstanceId" --output text)

  # Count the number of instance IDs
  num_instances=$(echo "$instance_ids" | wc -w)

  echo "Number of running EC2 instances: $num_instances"

  # Test passed if number of running instances is 1
  [ "$num_instances" -eq 0 ]

}

#!/usr/bin/env bats

INIT_CEDANA_CLI="$BATS_TEST_DIRNAME/../../cedana-cli"

@test "Cedana-cli executable exists" {
    [ -x "$INIT_CEDANA_CLI" ]
}

@test "Run job on instance" {
    run ./cedana-cli run job.yml > $BATS_TMPDIR/log_output.txt

    # Test passed if success signal is received
    [ "$status" -eq 0 ]
}


@test "Tear down all instances" {
  run ./cedana-cli destroy-all > $BATS_TMPDIR/log_output.txt

  # Test passed if success signal is received
  [ "$status" -eq 0 ]
}

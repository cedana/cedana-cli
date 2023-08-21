# cedana-cli integration tests

Integration tests provide end-to-end testing of cedana and do _not_ replace unit tests. Integration tests should be written w/ bash in bats (https://github.com/bats-core/bats-core).

# How to use:

1. The main script is test.bats, to run it: bats ./test/integration/test.bats
2. Make sure you have bats installed:
   git clone https://github.com/bats-core/bats-core.git
   cd bats-core
   ./install.sh /usr/local
   done!
3. For full integration tests, you will need aws cli installed and configured with you IAM that is associated with Cedana's AWS org.
   You will also need cedana-cli bootstrapped with the configuration settings of heartbeat_enabled and keep_running set to true.
   You will also need an example job.yml in /test/integration/jobs. Check the test.bats file for more info.

```
instance_specs:
  memory_gb: 12
  cpu_cores: 2
  max_price_usd_hour: 0.2
work_dir: "../../../benchmarking/processes/loop"
task:
  run:
    - "./loop"
restored_task:
  run:
    - ""

```

Above is a very basic example job.yml

<h3>Current structure of tests</h3>
Currently there are 9 tests, the first 5 tests are preliminary tests for items that will fail future tests. The last 4 tests are integration tests that check for the following:

1. A successful completion of _cedana-cli run_ command.
2. Checks aws for the specifc ec2 instance and if it's running.
3. A successful completion of _cedana-cli destroy-all_ command.
4. Checks aws for 0 instances running after destroy command.

Future checks:

- Verify # of nats publishes vs expected:
  For this we are pushing to instances.db the commands and the # of commands. Then checking ACK count, to avoid desync, + 1 difference is okay for now.
- Verify specific ec2 instance if running or destroyed based off top(1) or 2 of instances and job db

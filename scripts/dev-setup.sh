#!/bin/sh
# create a dev config in .cedana if it doesn't exist
DEV_CONFIG="{ \"self_serve\": false, \"enabled_providers\": [ \"aws\" ], \"shared_storage\": { \"mount_point\": \"\/home\/nravic\/.cedana\/\", \"dump_storage_dir\": \"\/home\/nravic\/.cedana\/\" }, \"checkpoint\": { \"heartbeat_enabled\": false, \"heartbeat_interval_seconds\": 60 }, \"connection\": { \"nats_url\": \"0.0.0.0\", \"nats_port\": 4222, \"auth_token\": \"test\" } }"

if [ ! -f ~/.cedana/cedana_config_dev.json ]; then
   echo "Creating ~/.cedana/cedana_config_dev.json"
   echo $DEV_CONFIG > ~/.cedana/cedana_config_dev.json
fi

# start simple loop for checkpointing 
## setup cedana-cli for testing 
auth_token="test"

DEV_CONFIG="{ \"self_serve\": false, \"enabled_providers\": [ \"local\" ], \"shared_storage\": { \"mount_point\": \"\/home\/nravic\/.cedana\/\", \"dump_storage_dir\": \"\/home\/nravic\/.cedana\/\" }, \"checkpoint\": { \"heartbeat_enabled\": false, \"heartbeat_interval_seconds\": 60 }, \"connection\": { \"nats_url\": \"0.0.0.0\", \"nats_port\": 4222, \"auth_token\": \"$auth_token\" } }"

# create folder if it doesnt exist 
if [ ! -d ~/.cedana ]; then
   mkdir ~/.cedana
fi 

# move existing config to backup 
if [ -f ~/.cedana/cedana_config.json ]; then
   mv ~/.cedana/cedana_config.json ~/.cedana/cedana_config.bak.json
fi

echo "Creating developer ~/.cedana/cedana_config.json"
echo $DEV_CONFIG > ~/.cedana/cedana_config.json


## start cedana-cli 
.././cedana-cli debug setup_test devjob devclient
.././cedana-cli debug create_dev_instance devclient
.././cedana-cli daemon -o devorch -j devjob -c devclient 

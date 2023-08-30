#!/bin/sh
sudo docker run -p 4222:4222 -ti nats:latest -js --auth test 
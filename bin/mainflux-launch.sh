#!/bin/bash

###
# Launches all Mainflux Go binaries when they are installed globally.
# Also launches NATS broker instance, expecting that
# `gnatsd` is installed globally.
#
# Expects that Cassandra is already installed and running.
#
# Does not launch NodeJS MQTT service - this one must be launched by hand for now.
###

# Kill all mainflux-* stuff
function cleanup {
	pkill mainflux
}

gnatsd &
# Wait a bit for NATS to be on
sleep 0.1
mainflux-http &
mainflux-manager &
mainflux-writer &
mainflux-coap &

trap cleanup EXIT

while : ; do sleep 1 ; done



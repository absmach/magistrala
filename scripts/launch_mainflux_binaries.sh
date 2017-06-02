#!/bin/bash

###
# Launches all Mainflux Go binaries
# when they are installed globally.
#
# Expects that influxDB and MongoDB are already installed and running.
#
# Does not launch NodeJS MQTT service - this one must be launched by hand for now.
###

# Kil all mainflux-* stuff
function cleanup {
	pkill mainflux
}

gnatsd &
mainflux-http-sender &
mainflux-influxdb-writer &
mainflux-influxdb-reader &
mainflux-manager &

trap cleanup EXIT

while : ; do sleep 1 ; done



#!/bin/bash

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



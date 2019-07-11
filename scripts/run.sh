#!/bin/bash
#
# Copyright (c) 2018
# Mainflux
#
# SPDX-License-Identifier: Apache-2.0
#

###
# Runs all Mainflux microservices (must be previously built and installed).
#
# Expects that PostgreSQL and needed messaging DB are alredy running.
# Additionally, MQTT microservice demands that Redis is up and running.
#
###

BUILD_DIR=../build

# Kill all mainflux-* stuff
function cleanup {
    pkill mainflux
    pkill nats
}

###
# NATS
###
gnatsd &

###
# Users
###
MF_USERS_LOG_LEVEL=info $BUILD_DIR/mainflux-users &

###
# Things
###
MF_THINGS_LOG_LEVEL=info MF_THINGS_HTTP_PORT=8182 MF_THINGS_AUTH_GRPC_PORT=8183 MF_THINGS_AUTH_HTTP_PORT=8989 $BUILD_DIR/mainflux-things &

###
# HTTP
###
MF_HTTP_ADAPTER_LOG_LEVEL=info MF_HTTP_ADAPTER_PORT=8185 MF_THINGS_URL=localhost:8183 $BUILD_DIR/mainflux-http &

###
# WS
###
MF_WS_ADAPTER_LOG_LEVEL=info MF_WS_ADAPTER_PORT=8186 MF_THINGS_URL=localhost:8183 $BUILD_DIR/mainflux-ws &

###
# NORMALIZER 
###
MF_NORMALIZER_LOG_LEVEL=INFO MF_NORMALIZER_PORT=8184 MF_NATS_URL=localhost:4222 $BUILD_DIR/mainflux-normalizer &

###
# MQTT
###
# Switch to top dir to find *.proto stuff when running MQTT broker

cd ..
MF_MQTT_ADAPTER_LOG_LEVEL=info MF_THINGS_URL=localhost:8183 node mqtt/mqtt.js &
cd -

###
# CoAP
###
MF_COAP_ADAPTER_LOG_LEVEL=info MF_COAP_ADAPTER_PORT=5683 MF_THINGS_URL=localhost:8183 $BUILD_DIR/mainflux-coap &

trap cleanup EXIT

while : ; do sleep 1 ; done

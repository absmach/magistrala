#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

###
# Runs all SuperMQ microservices (must be previously built and installed).
#
# Expects that PostgreSQL and needed messaging DB are alredy running.
# Additionally, MQTT microservice demands that Redis is up and running.
#
###

BUILD_DIR=../build

# Kill all supermq-* stuff
function cleanup {
    pkill supermq
    pkill nats
}

###
# NATS
###
nats-server &
counter=1
until fuser 4222/tcp 1>/dev/null 2>&1;
do
    sleep 0.5
    ((counter++))
    if [ ${counter} -gt 10 ]
    then
        echo "NATS failed to start in 5 sec, exiting"
        exit 1
    fi
    echo "Waiting for NATS server"
done

###
# Users
###
SMQ_USERS_LOG_LEVEL=info SMQ_USERS_HTTP_PORT=9002 SMQ_USERS_GRPC_PORT=7001 SMQ_USERS_ADMIN_EMAIL=admin@supermq.com SMQ_USERS_ADMIN_PASSWORD=12345678 SMQ_USERS_ADMIN_USERNAME=admin SMQ_EMAIL_TEMPLATE=../docker/templates/users.tmpl $BUILD_DIR/supermq-users &

###
# Clients
###
SMQ_CLIENTS_LOG_LEVEL=info SMQ_CLIENTS_HTTP_PORT=9000 SMQ_CLIENTS_GRPC_PORT=7000 SMQ_CLIENTS_AUTH_HTTP_PORT=9002 $BUILD_DIR/supermq-clients &

###
# HTTP
###
SMQ_HTTP_ADAPTER_LOG_LEVEL=info SMQ_HTTP_ADAPTER_PORT=8008 SMQ_CLIENTS_GRPC_URL=localhost:7000 $BUILD_DIR/supermq-http &

###
# WS
###
SMQ_WS_ADAPTER_LOG_LEVEL=info SMQ_WS_ADAPTER_HTTP_PORT=8190 SMQ_CLIENTS_GRPC_URL=localhost:7000 $BUILD_DIR/supermq-ws &

###
# MQTT
###
SMQ_MQTT_ADAPTER_LOG_LEVEL=info SMQ_CLIENTS_GRPC_URL=localhost:7000 $BUILD_DIR/supermq-mqtt &

###
# CoAP
###
SMQ_COAP_ADAPTER_LOG_LEVEL=info SMQ_COAP_ADAPTER_PORT=5683 SMQ_CLIENTS_GRPC_URL=localhost:7000 $BUILD_DIR/supermq-coap &

trap cleanup EXIT

while : ; do sleep 1 ; done

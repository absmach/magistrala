#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

###
# Runs all Magistrala microservices (must be previously built and installed).
#
# Expects that PostgreSQL and needed messaging DB are alredy running.
# Additionally, MQTT microservice demands that Redis is up and running.
#
###

BUILD_DIR=../build

# Kill all magistrala-* stuff
function cleanup {
    pkill magistrala
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
MG_USERS_LOG_LEVEL=info MG_USERS_HTTP_PORT=9002 MG_USERS_GRPC_PORT=7001 MG_USERS_ADMIN_EMAIL=admin@magistrala.com MG_USERS_ADMIN_PASSWORD=12345678 MG_USERS_ADMIN_USERNAME=admin MG_EMAIL_TEMPLATE=../docker/templates/users.tmpl $BUILD_DIR/magistrala-users &

###
# Clients
###
MG_CLIENTS_LOG_LEVEL=info MG_CLIENTS_HTTP_PORT=9000 MG_CLIENTS_AUTH_GRPC_PORT=7000 MG_CLIENTS_AUTH_HTTP_PORT=9002 $BUILD_DIR/magistrala-clients &

###
# HTTP
###
MG_HTTP_ADAPTER_LOG_LEVEL=info MG_HTTP_ADAPTER_PORT=8008 MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 $BUILD_DIR/magistrala-http &

###
# WS
###
MG_WS_ADAPTER_LOG_LEVEL=info MG_WS_ADAPTER_HTTP_PORT=8190 MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 $BUILD_DIR/magistrala-ws &

###
# MQTT
###
MG_MQTT_ADAPTER_LOG_LEVEL=info MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 $BUILD_DIR/magistrala-mqtt &

###
# CoAP
###
MG_COAP_ADAPTER_LOG_LEVEL=info MG_COAP_ADAPTER_PORT=5683 MG_CLIENTS_AUTH_GRPC_URL=localhost:7000 $BUILD_DIR/magistrala-coap &

trap cleanup EXIT

while : ; do sleep 1 ; done

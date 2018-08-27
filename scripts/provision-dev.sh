#!/usr/bin/env bash
#
# Copyright (c) 2018
# Mainflux
#
# SPDX-License-Identifier: Apache-2.0
#

###
# Provisions example user, device and channel on a clean Mainflux installation.
#
# Expects a running Mainflux installation.
#
#
###

if [ $# -lt 4 ]
  then
    echo "Usage: $0 user_email user_password device_name channel_name"
    exit 1
fi

EMAIL=$1
PASSWORD=$2
DEVICE=$3
CHANNEL=$4

#provision user:
printf "Provisoning user with email $EMAIL and password $PASSWORD \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/users -d '{"email":"'"$EMAIL"'", "password":"'"$PASSWORD"'"}'

#get jwt token
JWTTOKEN=$(curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/tokens -d '{"email":"'"$EMAIL"'", "password":"'"$PASSWORD"'"}' | grep -Po "token\":\"\K(.*)(?=\")")
printf "JWT TOKEN for user is $JWTTOKEN \n"

#provision device
printf "Provisioning device with name $DEVICE \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: $JWTTOKEN" https://localhost/things -d '{"type":"device", "name":"'"$DEVICE"'"}'

#get device token
DEVICETOKEN=$(curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: $JWTTOKEN" https://localhost/things/1 | grep -Po "key\":\"\K(.*)(?=\")")
printf "Device token is $DEVICETOKEN \n"

#provision channel
printf "Provisioning channel with name $CHANNEL \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: $JWTTOKEN" https://localhost/channels -d '{"name":"'"$CHANNEL"'"}'

#connect device to channel
printf "Connecting device to channel \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X PUT -H "Authorization: $JWTTOKEN" https://localhost/channels/1/things/1

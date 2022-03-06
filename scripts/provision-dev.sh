#!/usr/bin/env bash
#
# Copyright (c) Mainflux
# SPDX-License-Identifier: Apache-2.0
#

###
# Provisions example user, thing and channel on a clean Mainflux installation.
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

#provision thing
printf "Provisioning thing with name $DEVICE \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/things -d '{"name":"'"$DEVICE"'"}'

#get thing token
DEVICETOKEN=$(curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -H "Authorization: Bearer $JWTTOKEN" https://localhost/things/1 | grep -Po "key\":\"\K(.*)(?=\")")
printf "Device token is $DEVICETOKEN \n"

#provision channel
printf "Provisioning channel with name $CHANNEL \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/channels -d '{"name":"'"$CHANNEL"'"}'

#connect thing to channel
printf "Connecting thing to channel \n"
curl -s -S --cacert docker/ssl/certs/mainflux-server.crt --insecure -X PUT -H "Authorization: Bearer $JWTTOKEN" https://localhost/channels/1/things/1

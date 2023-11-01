#!/usr/bin/env bash
#
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0
#

###
# Provisions example user, thing and channel on a clean Magistrala installation.
#
# Expects a running Magistrala installation.
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
curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/users -d '{"credentials": {"identity": "'"$EMAIL"'","secret": "'"$PASSWORD"'"}, "status": "enabled", "role": "admin"  }'

#get jwt token
JWTTOKEN=$(curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X POST -H "Content-Type: application/json" https://localhost/users/tokens/issue -d '{"identity":"'"$EMAIL"'", "secret":"'"$PASSWORD"'"}' | grep -oP '"access_token":"\K[^"]+' )
printf "JWT TOKEN for user is $JWTTOKEN \n"

#provision thing
printf "Provisioning thing with name $DEVICE \n"
DEVICEID=$(curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/things -d '{"name":"'"$DEVICE"'", "status": "enabled"}' | grep -oP '"id":"\K[^"]+' )
curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X GET -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/things/$DEVICEID

#get thing token
DEVICETOKEN=$(curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -H "Authorization: Bearer $JWTTOKEN" https://localhost/things/$DEVICEID | grep -oP '"secret":"\K[^"]+' )
printf "Device token is $DEVICETOKEN \n"

#provision channel
printf "Provisioning channel with name $CHANNEL \n"
CHANNELID=$(curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X POST -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/channels -d '{"name":"'"$CHANNEL"'", "status": "enabled"}' |  grep -oP '"id":"\K[^"]+' )
curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X GET -H "Content-Type: application/json" -H "Authorization: Bearer $JWTTOKEN" https://localhost/channels/$CHANNELID

#connect thing to channel
printf "Connecting thing of id $DEVICEID to channel of id $CHANNELID \n"
curl -s -S --cacert docker/ssl/certs/magistrala-server.crt --insecure -X PUT -H "Authorization: Bearer $JWTTOKEN" https://localhost/channels/$CHANNELID/things/$DEVICEID

#!/bin/bash

###
# Stops services on the local system prior to sunning containers
# in order to liberate binding ports
###

sudo service mongodb stop
sudo service influxdb stop
sudo service mosquitto stop
sudo service nginx stop

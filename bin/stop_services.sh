#!/bin/bash

###
# Stops services on the local system prior to running containers
# in order to liberate binding ports
###

sudo service mosquitto stop
sudo service nginx stop

#!/bin/bash

DIR=$PWD

mkdir -p ./mainflux
cd ./mainflux

if [ -z "$GOPATH" ]; then
	mkdir -p $PWD/go
	export GOPATH=$PWD/go
fi

export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

# Core
go get -v github.com/mainflux/mainflux-core

# Auth
go get -v github.com/mainflux/mainflux-auth

# Cli
go get -v github.com/mainflux/mainflux-cli

# MQTT
git clone https://github.com/mainflux/mainflux-mqtt
cd mainflux-mqtt
npm install
cd ..

# NGINX Conf
git clone https://github.com/mainflux/mainflux-nginx

# NATS
go get -v github.com/nats-io/gnatsd

# Make symlink to go mainflux sources
ln -s $GOPATH/src/github.com/mainflux mainflux-go

# Go back to where we started
cd $DIR

# Print info
cat << EOF

***

# Mainflux is now installed #

- Go sources are located at $GOPATH/src
- Go binaries are located are $GOBIN
- MQTT NodeJS sources are located at $PWD/mainflux/mainflux-mqtt
- NGINX config files are located  in $PWD/mainflux/mainflux-nginx

External dependencies needed for Mainflux are:
- MongoDB
- NATS
- Redis
- NGINX

NATS have been installed, for MongoDB, Redis and NGINX
run something like:

sudo apt-get install mongodb redis-server nginx

NGINX config has been cloned in mainflux-nginx,
and these config files have to be copied to /etc/nginx once NGINX server
is installed on the system.
After copying these files you have to re-start the nginx service:

sudo systemctl restart nginx.service

***

EOF


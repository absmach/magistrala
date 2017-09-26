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
mkdir -p $GOBIN

# Mainflux Go microservices
go get -d -v github.com/mainflux/mainflux
cd $GOPATH/src/github.com/mainflux/mainflux
make
make install
cd -


# MQTT
git clone https://github.com/mainflux/mqtt-adapter
cd mqtt-adapter
npm install
cd ..

# NGINX Conf
git clone https://github.com/mainflux/proxy

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
- MQTT NodeJS sources are located at $PWD/mainflux/mqtt-adapter
- NGINX config files are located  in $PWD/mainflux/nginx-conf

External dependencies needed for Mainflux are:
- Cassandra
- NATS
- NGINX

NATS have been installed.
For Cassandra follow the instructions at http://cassandra.apache.org/download/
For NGINX follow the instructions here: http://nginx.org/en/docs/install.html

NGINX config has been cloned in nginx-conf,
and these config files have to be copied to /etc/nginx once NGINX server
is installed on the system.
After copying these files you have to re-start the nginx service:

sudo systemctl restart nginx.service

***

EOF


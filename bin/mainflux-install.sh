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

NATS has been installed.
For Cassandra follow the instructions at http://cassandra.apache.org/download/

After installing Cassandra you should create the two keyspaces that Mainflux uses. This can be done with something similar to:

cqlsh> CREATE KEYSPACE IF NOT EXISTS manager WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
cqlsh> CREATE KEYSPACE IF NOT EXISTS message_writer WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };

Please note that in the example SQL statment above, the keyspaces will be created in a single datacenter (single cluster) and there will
only be one replica (copy) of the data. You should create the keyspaces with parameters appropriate for your Cassandra installation. Take a look
at the Cassandra documentation for creating keyspaces for more details. For production usage you should always configure multiple replicas in order
to have data redundancy and be safe in case one or more Cassandra cluster nodes fail.


For NGINX follow the instructions here: http://nginx.org/en/docs/install.html

NGINX config has been cloned in nginx-conf,
and these config files have to be copied to /etc/nginx once NGINX server
is installed on the system.
After copying these files you have to re-start the nginx service:

sudo systemctl restart nginx.service

***

EOF

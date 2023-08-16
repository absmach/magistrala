#!/usr/bin/env bash
until printf "" 2>>/dev/null >>/dev/tcp/mainflux-cassandra/9042; do
    sleep 5;
    echo "Waiting for cassandra...";
done

echo "Creating keyspace and table..."
cqlsh mainflux-cassandra  -e "CREATE KEYSPACE IF NOT EXISTS mainflux WITH replication = {'class':'SimpleStrategy','replication_factor':'1'};"

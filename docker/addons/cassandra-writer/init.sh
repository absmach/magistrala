#!/usr/bin/env bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

until printf "" 2>>/dev/null >>/dev/tcp/magistrala-cassandra/9042; do
    sleep 5;
    echo "Waiting for cassandra...";
done

echo "Creating keyspace and table..."
cqlsh magistrala-cassandra  -e "CREATE KEYSPACE IF NOT EXISTS magistrala WITH replication = {'class':'SimpleStrategy','replication_factor':'1'};"

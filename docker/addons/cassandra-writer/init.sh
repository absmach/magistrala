docker-compose -f docker/addons/cassandra-writer/docker-compose.yml up -d
sleep 20
docker exec mainflux-cassandra cqlsh -e "CREATE KEYSPACE IF NOT EXISTS mainflux WITH replication = {'class':'SimpleStrategy','replication_factor':'1'};"

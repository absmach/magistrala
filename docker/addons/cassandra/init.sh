docker-compose -f docker/docker-compose.yml -f docker/addons/cassandra/docker-compose.yml up -d
sleep 20
docker exec mainflux-cassandra cqlsh -e "CREATE KEYSPACE IF NOT EXISTS mainflux WITH replication = {'class':'SimpleStrategy','replication_factor':'1'};"

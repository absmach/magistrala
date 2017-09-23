package cassandra

import "github.com/gocql/gocql"

var tables []string = []string{
	`CREATE TABLE IF NOT EXISTS users (
		email text,
		password text,
		PRIMARY KEY (email)
	)`,
	`CREATE TABLE IF NOT EXISTS clients_by_user (
		user text,
		id timeuuid,
		type text,
		name text,
		access_key text,
		meta map<text, text>,
		PRIMARY KEY ((user), id)
	)`,
	`CREATE TABLE IF NOT EXISTS channels_by_user (
		user text,
		id timeuuid,
		name text,
		connected set<text>,
		PRIMARY KEY ((user), id)
	)`,
	`CREATE MATERIALIZED VIEW IF NOT EXISTS clients_by_channel
		AS SELECT user, id, connected FROM channels_by_user
		WHERE id IS NOT NULL
		PRIMARY KEY (id, user)
	`,
}

// Connect establishes connection to the Cassandra cluster.
func Connect(hosts []string, keyspace string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum

	return cluster.CreateSession()
}

// Initialize creates tables used by the service.
func Initialize(session *gocql.Session) error {
	for _, table := range tables {
		if err := session.Query(table).Exec(); err != nil {
			return err
		}
	}

	return nil
}

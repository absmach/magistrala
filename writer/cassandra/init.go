package cassandra

import "github.com/gocql/gocql"

var tables []string = []string{
	`CREATE TABLE IF NOT EXISTS messages_by_channel (
		channel timeuuid,
		id timeuuid,
		publisher text,
		protocol text,
		bn text,
		bt double,
		bu text,
		bv double,
		bs double,
		bver int,
		n text,
		u text,
		v double,
		vs text,
		vb boolean,
		vd text,
		s double,
		t double,
		ut double,
		l text,
		PRIMARY KEY ((channel), id)
	) WITH default_time_to_live = 86400`,
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

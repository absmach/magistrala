package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.ClientRepository = (*clientRepository)(nil)

type clientRepository struct {
	session *gocql.Session
}

// NewClientRepository instantiates Cassandra client repository.
func NewClientRepository(session *gocql.Session) manager.ClientRepository {
	return &clientRepository{session}
}

func (repo *clientRepository) Id() string {
	return gocql.TimeUUID().String()
}

func (repo *clientRepository) Save(client manager.Client) error {
	cql := `INSERT INTO clients_by_user (user, id, type, name, access_key, meta)
		VALUES (?, ?, ?, ?, ?, ?)`

	if err := repo.session.Query(cql, client.Owner, client.ID,
		client.Type, client.Name, client.Key, client.Meta).Exec(); err != nil {
		return err
	}

	return nil
}

func (repo *clientRepository) Update(client manager.Client) error {
	cql := `UPDATE clients_by_user SET type = ?, name = ?, meta = ?
		WHERE user = ? AND id = ? IF EXISTS`

	applied, err := repo.session.Query(cql, client.Type, client.Name, client.Meta,
		client.Owner, client.ID).ScanCAS()

	if !applied {
		return manager.ErrNotFound
	}

	return err
}

func (repo *clientRepository) One(owner string, id string) (manager.Client, error) {
	cql := `SELECT type, name, access_key, meta FROM clients_by_user
		WHERE user = ? AND id = ? LIMIT 1`

	cli := manager.Client{
		Owner: owner,
		ID:    id,
	}

	if err := repo.session.Query(cql, owner, id).
		Scan(&cli.Type, &cli.Name, &cli.Key, &cli.Meta); err != nil {
		return cli, manager.ErrNotFound
	}

	return cli, nil
}

func (repo *clientRepository) All(owner string) []manager.Client {
	cql := `SELECT id, type, name, access_key, meta FROM clients_by_user WHERE user = ?`
	var id string
	var cType string
	var name string
	var key string
	var meta map[string]string

	// NOTE: the closing might failed
	iter := repo.session.Query(cql, owner).Iter()
	defer iter.Close()

	clients := make([]manager.Client, 0)

	for iter.Scan(&id, &cType, &name, &key, &meta) {
		c := manager.Client{
			Owner: owner,
			ID:    id,
			Type:  cType,
			Name:  name,
			Key:   key,
			Meta:  meta,
		}

		clients = append(clients, c)
	}

	return clients
}

func (repo *clientRepository) Remove(owner string, id string) error {
	cql := `DELETE FROM clients_by_user WHERE user = ? AND id = ?`
	return repo.session.Query(cql, owner, id).Exec()
}

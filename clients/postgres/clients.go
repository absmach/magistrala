package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/logger"
	uuid "github.com/satori/go.uuid"
)

var _ clients.ClientRepository = (*clientRepository)(nil)

type clientRepository struct {
	db  *sql.DB
	log logger.Logger
}

// NewClientRepository instantiates a PostgreSQL implementation of client
// repository.
func NewClientRepository(db *sql.DB, log logger.Logger) clients.ClientRepository {
	return &clientRepository{db: db, log: log}
}

func (cr clientRepository) ID() string {
	return uuid.NewV4().String()
}

func (cr clientRepository) Save(client clients.Client) error {
	q := `INSERT INTO clients (id, owner, type, name, key, payload) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := cr.db.Exec(q, client.ID, client.Owner, client.Type, client.Name, client.Key, client.Payload)
	return err
}

func (cr clientRepository) Update(client clients.Client) error {
	q := `UPDATE clients SET name = $1, payload = $2 WHERE owner = $3 AND id = $4;`

	res, err := cr.db.Exec(q, client.Name, client.Payload, client.Owner, client.ID)
	if err != nil {
		return err
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if cnt == 0 {
		return clients.ErrNotFound
	}

	return nil
}

func (cr clientRepository) One(owner, id string) (clients.Client, error) {
	q := `SELECT name, type, key, payload FROM clients WHERE id = $1 AND owner = $2`
	client := clients.Client{ID: id, Owner: owner}
	err := cr.db.
		QueryRow(q, id, owner).
		Scan(&client.Name, &client.Type, &client.Key, &client.Payload)

	if err != nil {
		empty := clients.Client{}
		if err == sql.ErrNoRows {
			return empty, clients.ErrNotFound
		}
		return empty, err
	}

	return client, nil
}

func (cr clientRepository) All(owner string, offset, limit int) []clients.Client {
	q := `SELECT id, name, type, key, payload FROM clients WHERE owner = $1 LIMIT $2 OFFSET $3`
	items := []clients.Client{}

	rows, err := cr.db.Query(q, owner, limit, offset)
	if err != nil {
		cr.log.Error(fmt.Sprintf("Failed to retrieve clients due to %s", err))
		return []clients.Client{}
	}
	defer rows.Close()

	for rows.Next() {
		c := clients.Client{Owner: owner}
		if err = rows.Scan(&c.ID, &c.Name, &c.Type, &c.Key, &c.Payload); err != nil {
			cr.log.Error(fmt.Sprintf("Failed to read retrieved client due to %s", err))
			return []clients.Client{}
		}
		items = append(items, c)
	}

	return items
}

func (cr clientRepository) Remove(owner, id string) error {
	q := `DELETE FROM clients WHERE id = $1 AND owner = $2`
	cr.db.Exec(q, id, owner)
	return nil
}

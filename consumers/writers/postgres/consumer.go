// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/pkg/errors"
	mgjson "github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errSaveMessage    = errors.New("failed to save message to postgres database")
	errTransRollback  = errors.New("failed to rollback transaction")
	errNoTable        = errors.New("relation does not exist")
)

var _ consumers.BlockingConsumer = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) consumers.BlockingConsumer {
	return &postgresRepo{db: db}
}

func (pr postgresRepo) ConsumeBlocking(ctx context.Context, message interface{}) (err error) {
	switch m := message.(type) {
	case mgjson.Messages:
		return pr.saveJSON(ctx, m)
	default:
		return pr.saveSenml(ctx, m)
	}
}

func (pr postgresRepo) saveSenml(ctx context.Context, messages interface{}) (err error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	q := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:id, :channel, :subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	tx, err := pr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}
	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(errTransRollback, txErr))
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(errSaveMessage, err)
		}
	}()

	for _, msg := range msgs {
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		m := senmlMessage{Message: msg, ID: id.String()}
		if _, err := tx.NamedExec(q, m); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				if pgErr.Code == pgerrcode.InvalidTextRepresentation {
					return errors.Wrap(errSaveMessage, errInvalidMessage)
				}
			}

			return errors.Wrap(errSaveMessage, err)
		}
	}
	return err
}

func (pr postgresRepo) saveJSON(ctx context.Context, msgs mgjson.Messages) error {
	if err := pr.insertJSON(ctx, msgs); err != nil {
		if err == errNoTable {
			if err := pr.createTable(msgs.Format); err != nil {
				return err
			}
			return pr.insertJSON(ctx, msgs)
		}
		return err
	}
	return nil
}

func (pr postgresRepo) insertJSON(ctx context.Context, msgs mgjson.Messages) error {
	tx, err := pr.db.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}
	defer func() {
		if err != nil {
			if txErr := tx.Rollback(); txErr != nil {
				err = errors.Wrap(err, errors.Wrap(errTransRollback, txErr))
			}
			return
		}

		if err = tx.Commit(); err != nil {
			err = errors.Wrap(errSaveMessage, err)
		}
	}()

	q := `INSERT INTO %s (id, channel, created, subtopic, publisher, protocol, payload)
          VALUES (:id, :channel, :created, :subtopic, :publisher, :protocol, :payload);`
	q = fmt.Sprintf(q, msgs.Format)

	for _, m := range msgs.Data {
		var dbmsg jsonMessage
		dbmsg, err = toJSONMessage(m)
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}

		if _, err = tx.NamedExec(q, dbmsg); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errSaveMessage, errInvalidMessage)
				case pgerrcode.UndefinedTable:
					return errNoTable
				}
			}
			return err
		}
	}
	return nil
}

func (pr postgresRepo) createTable(name string) error {
	q := `CREATE TABLE IF NOT EXISTS %s (
            id            UUID,
            created       BIGINT,
            channel       VARCHAR(254),
            subtopic      VARCHAR(254),
            publisher     VARCHAR(254),
            protocol      TEXT,
            payload       JSONB,
            PRIMARY KEY (id)
        )`
	q = fmt.Sprintf(q, name)

	_, err := pr.db.Exec(q)
	return err
}

type senmlMessage struct {
	senml.Message
	ID string `db:"id"`
}

type jsonMessage struct {
	ID        string `db:"id"`
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func toJSONMessage(msg mgjson.Message) (jsonMessage, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return jsonMessage{}, err
	}

	data := []byte("{}")
	if msg.Payload != nil {
		b, err := json.Marshal(msg.Payload)
		if err != nil {
			return jsonMessage{}, errors.Wrap(errSaveMessage, err)
		}
		data = b
	}

	m := jsonMessage{
		ID:        id.String(),
		Channel:   msg.Channel,
		Created:   msg.Created,
		Subtopic:  msg.Subtopic,
		Publisher: msg.Publisher,
		Protocol:  msg.Protocol,
		Payload:   data,
	}

	return m, nil
}

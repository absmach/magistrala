// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/pkg/errors"
	smqjson "github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errSaveMessage    = errors.New("failed to save message to postgres database")
	errTransRollback  = errors.New("failed to rollback transaction")
	errNoTable        = errors.New("relation does not exist")
	errNoColumn       = errors.New("column does not exist")
)

var _ consumers.BlockingConsumer = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) consumers.BlockingConsumer {
	return &postgresRepo{db: db}
}

func (pr postgresRepo) ConsumeBlocking(ctx context.Context, message any) (err error) {
	switch m := message.(type) {
	case smqjson.Messages:
		return pr.saveJSON(ctx, m)
	default:
		return pr.saveSenml(ctx, m)
	}
}

func (pr postgresRepo) saveSenml(ctx context.Context, messages any) (err error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	q := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time, device_id)
          VALUES (:id, :channel, :subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time, :device_id);`

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

// saveJSON writes a batch, creating or catching up the format's table on demand.
// JSON tables are created lazily and so are outside the migration set: a table
// created before device_id existed is missing the column, which surfaces as an
// undefined-column error on the first insert and is repaired the same way a
// missing table is.
func (pr postgresRepo) saveJSON(ctx context.Context, msgs smqjson.Messages) error {
	if err := pr.insertJSON(ctx, msgs); err != nil {
		if err == errNoTable || err == errNoColumn {
			if err := pr.ensureTable(msgs.Format); err != nil {
				return err
			}
			return pr.insertJSON(ctx, msgs)
		}
		return err
	}
	return nil
}

func (pr postgresRepo) insertJSON(ctx context.Context, msgs smqjson.Messages) error {
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

	q := `INSERT INTO %s (id, channel, created, subtopic, publisher, protocol, payload, device_id)
          VALUES (:id, :channel, :created, :subtopic, :publisher, :protocol, :payload, :device_id);`
	q = fmt.Sprintf(q, msgs.Format)

	for _, m := range msgs.Data {
		var dbmsg jsonMessage
		dbmsg, err = toJSONMessage(m)
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}

		if _, err = tx.NamedExec(q, dbmsg); err != nil {
			if preErr, ok := err.(*pgconn.PrepareError); ok {
				err = preErr.Unwrap()
			}
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errSaveMessage, errInvalidMessage)
				case pgerrcode.UndefinedTable:
					return errNoTable
				case pgerrcode.UndefinedColumn:
					return errNoColumn
				}
			}
			return err
		}
	}
	return nil
}

func (pr postgresRepo) ensureTable(name string) error {
	q := `CREATE TABLE IF NOT EXISTS %s (
            id            UUID,
            created       BIGINT,
            channel       VARCHAR(254),
            subtopic      VARCHAR(254),
            publisher     VARCHAR(254),
            protocol      TEXT,
            payload       JSONB,
            device_id     TEXT NOT NULL DEFAULT '',
            PRIMARY KEY (id)
        )`
	if _, err := pr.db.Exec(fmt.Sprintf(q, name)); err != nil {
		return err
	}

	// Catches up a table created before device_id existed.
	_, err := pr.db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT ''`, name))
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
	DeviceID  string `db:"device_id"`
}

func toJSONMessage(msg smqjson.Message) (jsonMessage, error) {
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
		DeviceID:  msg.DeviceId,
	}

	return m, nil
}

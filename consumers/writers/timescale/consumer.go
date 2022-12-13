// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	mfjson "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

var (
	errInvalidMessage = errors.New("invalid message representation")
	errSaveMessage    = errors.New("failed to save message to timescale database")
	errTransRollback  = errors.New("failed to rollback transaction")
	errNoTable        = errors.New("relation does not exist")
)

var _ consumers.Consumer = (*timescaleRepo)(nil)

type timescaleRepo struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) consumers.Consumer {
	return &timescaleRepo{db: db}
}

func (tr timescaleRepo) Consume(message interface{}) (err error) {
	switch m := message.(type) {
	case mfjson.Messages:
		return tr.saveJSON(m)
	default:
		return tr.saveSenml(m)
	}
}

func (tr timescaleRepo) saveSenml(messages interface{}) (err error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	q := `INSERT INTO messages (channel, subtopic, publisher, protocol,
          name, unit, value, string_value, bool_value, data_value, sum,
          time, update_time)
          VALUES (:channel, :subtopic, :publisher, :protocol, :name, :unit,
          :value, :string_value, :bool_value, :data_value, :sum,
          :time, :update_time);`

	tx, err := tr.db.BeginTxx(context.Background(), nil)
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
		m := senmlMessage{Message: msg}
		if _, err := tx.NamedExec(q, m); err != nil {
			pgErr, ok := err.(*pgconn.PgError)
			if ok {
				switch pgErr.Code {
				case pgerrcode.InvalidTextRepresentation:
					return errors.Wrap(errSaveMessage, errInvalidMessage)
				}
			}

			return errors.Wrap(errSaveMessage, err)
		}
	}
	return err
}

func (tr timescaleRepo) saveJSON(msgs mfjson.Messages) error {
	if err := tr.insertJSON(msgs); err != nil {
		if err == errNoTable {
			if err := tr.createTable(msgs.Format); err != nil {
				return err
			}
			return tr.insertJSON(msgs)
		}
		return err
	}
	return nil
}

func (tr timescaleRepo) insertJSON(msgs mfjson.Messages) error {
	tx, err := tr.db.BeginTxx(context.Background(), nil)
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

	q := `INSERT INTO %s (channel, created, subtopic, publisher, protocol, payload)
          VALUES (:channel, :created, :subtopic, :publisher, :protocol, :payload);`
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

func (tr timescaleRepo) createTable(name string) error {
	q := `CREATE TABLE IF NOT EXISTS %s (
            created       BIGINT NOT NULL,
            channel       VARCHAR(254),
            subtopic      VARCHAR(254),
            publisher     VARCHAR(254),
            protocol      TEXT,
            payload       JSONB,
            PRIMARY KEY (created, publisher, subtopic)
        );`
	q = fmt.Sprintf(q, name)

	_, err := tr.db.Exec(q)
	return err
}

type senmlMessage struct {
	senml.Message
}

type jsonMessage struct {
	Channel   string `db:"channel"`
	Created   int64  `db:"created"`
	Subtopic  string `db:"subtopic"`
	Publisher string `db:"publisher"`
	Protocol  string `db:"protocol"`
	Payload   []byte `db:"payload"`
}

func toJSONMessage(msg mfjson.Message) (jsonMessage, error) {
	data := []byte("{}")
	if msg.Payload != nil {
		b, err := json.Marshal(msg.Payload)
		if err != nil {
			return jsonMessage{}, errors.Wrap(errSaveMessage, err)
		}
		data = b
	}

	m := jsonMessage{
		Channel:   msg.Channel,
		Created:   msg.Created,
		Subtopic:  msg.Subtopic,
		Publisher: msg.Publisher,
		Protocol:  msg.Protocol,
		Payload:   data,
	}

	return m, nil
}

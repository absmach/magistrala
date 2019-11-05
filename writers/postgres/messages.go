// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux/transformers/senml"
	"github.com/mainflux/mainflux/writers"
)

const errInvalid = "invalid_text_representation"

// ErrInvalidMessage indicates that service received message that
// doesn't fit required format.
var ErrInvalidMessage = errors.New("invalid message representation")

var _ writers.MessageRepository = (*postgresRepo)(nil)

type postgresRepo struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) writers.MessageRepository {
	return &postgresRepo{db: db}
}

func (pr postgresRepo) Save(messages ...senml.Message) error {
	q := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
    name, unit, value, string_value, bool_value, data_value, sum,
    time, update_time, link)
    VALUES (:id, :channel, :subtopic, :publisher, :protocol, :name, :unit,
    :value, :string_value, :bool_value, :data_value, :sum,
    :time, :update_time, :link);`

	tx, err := pr.db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		dbth, err := toDBMessage(msg)
		if err != nil {
			return err
		}

		if _, err := tx.NamedExec(q, dbth); err != nil {
			pqErr, ok := err.(*pq.Error)
			if ok {
				switch pqErr.Code.Name() {
				case errInvalid:
					return ErrInvalidMessage
				}
			}

			return err
		}
	}

	return tx.Commit()
}

type dbMessage struct {
	ID          string   `db:"id"`
	Channel     string   `db:"channel"`
	Subtopic    string   `db:"subtopic"`
	Publisher   string   `db:"publisher"`
	Protocol    string   `db:"protocol"`
	Name        string   `db:"name"`
	Unit        string   `db:"unit"`
	Value       *float64 `db:"value"`
	StringValue *string  `db:"string_value"`
	BoolValue   *bool    `db:"bool_value"`
	DataValue   *string  `db:"data_value"`
	Sum         *float64 `db:"sum"`
	Time        float64  `db:"time"`
	UpdateTime  float64  `db:"update_time"`
	Link        string   `db:"link"`
}

func toDBMessage(msg senml.Message) (dbMessage, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return dbMessage{}, err
	}

	m := dbMessage{
		ID:         id.String(),
		Channel:    msg.Channel,
		Subtopic:   msg.Subtopic,
		Publisher:  msg.Publisher,
		Protocol:   msg.Protocol,
		Name:       msg.Name,
		Unit:       msg.Unit,
		Time:       msg.Time,
		UpdateTime: msg.UpdateTime,
		Link:       msg.Link,
		Sum:        msg.Sum,
	}

	switch {
	case msg.Value != nil:
		m.Value = msg.Value
	case msg.StringValue != nil:
		m.StringValue = msg.StringValue
	case msg.DataValue != nil:
		m.DataValue = msg.DataValue
	case msg.BoolValue != nil:
		m.BoolValue = msg.BoolValue
	}

	return m, nil
}

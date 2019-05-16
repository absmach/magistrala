//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"errors"

	"github.com/gofrs/uuid"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux"
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

func (pr postgresRepo) Save(msg mainflux.Message) error {
	q := `INSERT INTO messages (id, channel, subtopic, publisher, protocol,
    name, unit, value, string_value, bool_value, data_value, value_sum,
    time, update_time, link)
    VALUES (:id, :channel, :subtopic, :publisher, :protocol, :name, :unit,
    :value, :string_value, :bool_value, :data_value, :value_sum,
    :time, :update_time, :link);`

	dbth, err := toDBMessage(msg)
	if err != nil {
		return err
	}

	if _, err := pr.db.NamedExec(q, dbth); err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid:
				return ErrInvalidMessage
			}
		}

		return err
	}

	return nil
}

type dbMessage struct {
	ID          string   `db:"id"`
	Channel     string   `db:"channel"`
	Subtopic    string   `db:"subtopic"`
	Publisher   string   `db:"publisher"`
	Protocol    string   `db:"protocol"`
	Name        string   `db:"name"`
	Unit        string   `db:"unit"`
	FloatValue  *float64 `db:"value"`
	StringValue *string  `db:"string_value"`
	BoolValue   *bool    `db:"bool_value"`
	DataValue   *string  `db:"data_value"`
	ValueSum    *float64 `db:"value_sum"`
	Time        float64  `db:"time"`
	UpdateTime  float64  `db:"update_time"`
	Link        string   `db:"link"`
}

func toDBMessage(msg mainflux.Message) (dbMessage, error) {
	var floatVal, valSum *float64
	var strVal, dataVal *string
	var boolVal *bool

	switch msg.Value.(type) {
	case *mainflux.Message_FloatValue:
		v := msg.GetFloatValue()
		floatVal = &v
	case *mainflux.Message_StringValue:
		v := msg.GetStringValue()
		strVal = &v
	case *mainflux.Message_DataValue:
		v := msg.GetDataValue()
		dataVal = &v
	case *mainflux.Message_BoolValue:
		v := msg.GetBoolValue()
		boolVal = &v
	}

	if msg.GetValueSum() != nil {
		v := msg.GetValueSum().GetValue()
		valSum = &v
	}

	id, err := uuid.NewV4()
	if err != nil {
		return dbMessage{}, err
	}

	return dbMessage{
		ID:          id.String(),
		Channel:     msg.Channel,
		Subtopic:    msg.Subtopic,
		Publisher:   msg.Publisher,
		Protocol:    msg.Protocol,
		Name:        msg.Name,
		Unit:        msg.Unit,
		FloatValue:  floatVal,
		StringValue: strVal,
		BoolValue:   boolVal,
		DataValue:   dataVal,
		ValueSum:    valSum,
		Time:        msg.Time,
		UpdateTime:  msg.UpdateTime,
		Link:        msg.Link,
	}, nil
}

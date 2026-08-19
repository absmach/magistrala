// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/absmach/magistrala/readers/pgutil"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

const (
	messageFieldChannel      = "channel"
	messageFieldDeviceID     = "device_id"
	messageFieldDeviceIDs    = "device_ids"
	messageFieldScope        = "device_scope"
	scopeParamPublishers     = "scope_publishers"
	scopeParamSelfPublishers = "scope_self_publishers"
	scopeParamDeviceIDs      = "scope_device_ids"
	messageFieldName         = "name"
	messageFieldProtocol     = "protocol"
	messageFieldPublisher    = "publisher"
	messageFieldPublishers   = "publishers"
	messageFieldSubtopic     = "subtopic"
	messageFieldValue        = "value"
)

type postgresRepository struct {
	db *sqlx.DB
}

// New returns new PostgreSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &postgresRepository{
		db: db,
	}
}

func (tr postgresRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	order := "time"
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		order = "created"
		format = rpm.Format
	}
	cond := fmtCondition(chanID, rpm)

	// format is a request-supplied table name (readers/api/http/requests.go
	// restricts it to a bare identifier, but that check lives in a different
	// package and this call site must not rely on a caller having applied
	// it). Quoting it here is what actually stops it from being interpreted
	// as anything other than one identifier -- an unquoted %s would let it
	// close the FROM clause and append arbitrary SQL that every WHERE
	// predicate MG-08's authorization relies on sits downstream of.
	q := fmt.Sprintf(`SELECT * FROM %s
    WHERE %s ORDER BY %s DESC
	LIMIT :limit OFFSET :offset;`, pq.QuoteIdentifier(format), cond, order)

	params := map[string]any{
		messageFieldChannel:      chanID,
		"limit":                  rpm.Limit,
		"offset":                 rpm.Offset,
		messageFieldSubtopic:     rpm.Subtopic,
		messageFieldPublisher:    rpm.Publisher,
		messageFieldPublishers:   rpm.Publishers,
		messageFieldDeviceIDs:    rpm.DeviceIDs,
		scopeParamPublishers:     rpm.DeviceScope.Publishers(),
		scopeParamSelfPublishers: rpm.DeviceScope.SelfPublishers(),
		scopeParamDeviceIDs:      rpm.DeviceScope.Devices(),
		messageFieldName:         rpm.Name,
		messageFieldProtocol:     rpm.Protocol,
		messageFieldValue:        rpm.Value,
		"bool_value":             rpm.BoolValue,
		"string_value":           rpm.StringValue,
		"data_value":             rpm.DataValue,
		"from":                   rpm.From,
		"to":                     rpm.To,
	}
	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgutil.PgError(err); ok {
			switch pgErr.Code {
			case pgerrcode.UndefinedTable:
				return readers.MessagesPage{}, nil
			case pgerrcode.UndefinedColumn:
				if pgutil.IsLegacyJSONDeviceFilter(format, defTable, rpm, pgErr) {
					return pgutil.EmptyPage(rpm), nil
				}
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	page := pgutil.EmptyPage(rpm)
	switch format {
	case defTable:
		for rows.Next() {
			msg := senmlMessage{Message: senml.Message{}}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}

			page.Messages = append(page.Messages, msg.Message)
		}
	default:
		for rows.Next() {
			msg := jsonMessage{}
			if err := rows.StructScan(&msg); err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			m, err := msg.toMap()
			if err != nil {
				return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
			}
			page.Messages = append(page.Messages, m)
		}
	}

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, pq.QuoteIdentifier(format), cond)
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgutil.PgError(err); ok {
			if pgErr.Code == pgerrcode.UndefinedColumn && pgutil.IsLegacyJSONDeviceFilter(format, defTable, rpm, pgErr) {
				return pgutil.EmptyPage(rpm), nil
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return page, err
		}
	}
	page.Total = total

	return page, nil
}

// ListGatewayDevices implements readers.MessageRepository.
func (tr postgresRepository) ListGatewayDevices(chanID, publisherID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	return pgutil.DeviceStats(tr.db, defTable, chanID, messageFieldPublisher, publisherID, messageFieldDeviceID, true, rpm)
}

// ListDeviceGateways implements readers.MessageRepository.
func (tr postgresRepository) ListDeviceGateways(chanID, deviceID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	return pgutil.DeviceStats(tr.db, defTable, chanID, messageFieldDeviceID, deviceID, messageFieldPublisher, false, rpm)
}

func fmtCondition(chanID string, rpm readers.PageMetadata) string {
	condition := `channel = :channel`

	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return condition
	}
	if err := json.Unmarshal(meta, &query); err != nil {
		return condition
	}

	_, hasPublishers := query[messageFieldPublishers]

	for name := range query {
		switch name {
		case messageFieldPublisher:
			if hasPublishers {
				continue
			}
			condition = fmt.Sprintf(`%s AND %s = :%s`, condition, name, name)
		case messageFieldPublishers:
			condition = fmt.Sprintf(`%s AND %s = ANY(:%s)`, condition, messageFieldPublisher, messageFieldPublishers)
		case messageFieldDeviceIDs:
			condition = fmt.Sprintf(`%s AND %s = ANY(:%s)`, condition, messageFieldDeviceID, messageFieldDeviceIDs)
		case messageFieldScope:
			// OR of three legs, not AND: a device is named by publisher when it
			// publishes for itself and by device_id when a gateway relays for
			// it, and one row carries only one of the two identities meaningfully
			// — but which one device_id holds is not always knowable from its
			// value alone (R1: a device's own SenML base name can populate
			// device_id on its self-published rows exactly like a relay would).
			//
			//  1. publisher = ANY(self-published, non-gateway devices): trusted
			//     unconditionally. These devices are never anyone's relay, so any
			//     row they publish is their own, whatever device_id holds.
			//  2. device_id = '' AND publisher = ANY(all held devices, gateways
			//     included): the previous guard, kept for a held device flagged
			//     as a gateway (spec §8 A12) publishing its own telemetry with no
			//     device_id set at all.
			//  3. device_id = ANY(held devices' serials): gateway-relayed rows,
			//     named by the relayed device's serial rather than the relaying
			//     gateway's publisher identity.
			//
			// Leg 2 exists so a gateway-flagged publisher does NOT get leg 1's
			// unconditional trust: its relayed rows carry a real device_id for a
			// different device, and matching on publisher alone would readmit
			// every device it has ever relayed for — the leak device_id = ''
			// was introduced to close in the first place.
			condition = fmt.Sprintf(`%s AND (%s = ANY(:%s) OR (%s = '' AND %s = ANY(:%s)) OR %s = ANY(:%s))`,
				condition, messageFieldPublisher, scopeParamSelfPublishers,
				messageFieldDeviceID, messageFieldPublisher, scopeParamPublishers,
				messageFieldDeviceID, scopeParamDeviceIDs)
		case
			messageFieldSubtopic,
			messageFieldName,
			messageFieldProtocol:
			condition = fmt.Sprintf(`%s AND %s = :%s`, condition, name, name)
		case "v":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s AND value %s :value`, condition, comparator)
		case "vb":
			condition = fmt.Sprintf(`%s AND bool_value = :bool_value`, condition)
		case "vs":
			comparator := readers.ParseValueComparator(query)
			switch comparator {
			case "=":
				condition = fmt.Sprintf("%s AND string_value = :string_value ", condition)
			case ">":
				condition = fmt.Sprintf("%s AND string_value LIKE '%%' || :string_value || '%%' AND string_value <> :string_value", condition)
			case ">=":
				condition = fmt.Sprintf("%s AND string_value LIKE '%%' || :string_value || '%%'", condition)
			case "<=":
				condition = fmt.Sprintf("%s AND :string_value LIKE '%%' || string_value || '%%'", condition)
			case "<":
				condition = fmt.Sprintf("%s AND :string_value LIKE '%%' || string_value || '%%' AND string_value <> :string_value", condition)
			}
		case "vd":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s AND data_value %s :data_value`, condition, comparator)
		case "from":
			condition = fmt.Sprintf(`%s AND time >= :from`, condition)
		case "to":
			condition = fmt.Sprintf(`%s AND time < :to`, condition)
		}
	}
	return condition
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
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

func (msg jsonMessage) toMap() (map[string]any, error) {
	ret := map[string]any{
		"id":                  msg.ID,
		messageFieldChannel:   msg.Channel,
		"created":             msg.Created,
		messageFieldSubtopic:  msg.Subtopic,
		messageFieldPublisher: msg.Publisher,
		messageFieldProtocol:  msg.Protocol,
		"payload":             map[string]any{},
	}
	// Mirrors the `omitempty` on senml.Message.DeviceId: rows with no device —
	// direct publishers, or anything written before this column existed — are
	// reported exactly as they were before.
	if msg.DeviceID != "" {
		ret[messageFieldDeviceID] = msg.DeviceID
	}
	pld := make(map[string]any)
	if err := json.Unmarshal(msg.Payload, &pld); err != nil {
		return nil, err
	}
	ret["payload"] = pld
	return ret, nil
}

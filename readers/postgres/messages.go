// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var _ readers.MessageRepository = (*postgresRepository)(nil)

const (
	messageFieldChannel    = "channel"
	messageFieldDeviceID   = "device_id"
	messageFieldDeviceIDs  = "device_ids"
	messageFieldScope      = "device_scope"
	scopeParamPublishers   = "scope_publishers"
	scopeParamDeviceIDs    = "scope_device_ids"
	messageFieldName       = "name"
	messageFieldProtocol   = "protocol"
	messageFieldPublisher  = "publisher"
	messageFieldPublishers = "publishers"
	messageFieldSubtopic   = "subtopic"
	messageFieldValue      = "value"
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

	q := fmt.Sprintf(`SELECT * FROM %s
    WHERE %s ORDER BY %s DESC
	LIMIT :limit OFFSET :offset;`, format, cond, order)

	params := map[string]any{
		messageFieldChannel:    chanID,
		"limit":                rpm.Limit,
		"offset":               rpm.Offset,
		messageFieldSubtopic:   rpm.Subtopic,
		messageFieldPublisher:  rpm.Publisher,
		messageFieldPublishers: rpm.Publishers,
		messageFieldDeviceIDs:  rpm.DeviceIDs,
		scopeParamPublishers:   rpm.DeviceScope.Publishers(),
		scopeParamDeviceIDs:    rpm.DeviceScope.Devices(),
		messageFieldName:       rpm.Name,
		messageFieldProtocol:   rpm.Protocol,
		messageFieldValue:      rpm.Value,
		"bool_value":           rpm.BoolValue,
		"string_value":         rpm.StringValue,
		"data_value":           rpm.DataValue,
		"from":                 rpm.From,
		"to":                   rpm.To,
	}
	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok {
			switch pgErr.Code {
			case pgerrcode.UndefinedTable:
				return readers.MessagesPage{}, nil
			case pgerrcode.UndefinedColumn:
				if isLegacyJSONDeviceFilter(format, rpm, pgErr) {
					return emptyPage(rpm), nil
				}
			}
		}
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	page := emptyPage(rpm)
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

	q = fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, format, cond)
	rows, err = tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok {
			if pgErr.Code == pgerrcode.UndefinedColumn && isLegacyJSONDeviceFilter(format, rpm, pgErr) {
				return emptyPage(rpm), nil
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

func emptyPage(rpm readers.PageMetadata) readers.MessagesPage {
	return readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}
}

// ListGatewayDevices implements readers.MessageRepository.
func (tr postgresRepository) ListGatewayDevices(chanID, publisherID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	return tr.deviceStats(chanID, messageFieldPublisher, publisherID, messageFieldDeviceID, true, rpm)
}

// ListDeviceGateways implements readers.MessageRepository.
func (tr postgresRepository) ListDeviceGateways(chanID, deviceID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	return tr.deviceStats(chanID, messageFieldDeviceID, deviceID, messageFieldPublisher, false, rpm)
}

// deviceStats implements both directions of the MG-15 observed-device
// aggregation: distinct values of groupCol, for rows on channel chanID with
// filterCol = filterVal, each with its last-seen time and message count.
//
// scoped applies DeviceScope to narrow groupCol itself. It is only ever true
// for the gateway->devices direction — see the DeviceScope comment on
// ListDeviceGateways in readers/messages.go for why the inverse direction
// does not narrow by scope at all.
func (tr postgresRepository) deviceStats(chanID, filterCol, filterVal, groupCol string, scoped bool, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	conditions := []string{
		fmt.Sprintf("%s = :channel", messageFieldChannel),
		fmt.Sprintf("%s = :filter_val", filterCol),
	}
	if groupCol == messageFieldDeviceID {
		// A publisher's direct (non-relayed) messages carry no device_id at
		// all; grouping without this would surface a spurious "" roster
		// entry for them. publisher, by contrast, is a required UUID column
		// that is never empty, so this guard only applies on this side —
		// comparing it against '' would also fail to parse as a UUID.
		conditions = append(conditions, fmt.Sprintf("%s <> ''", groupCol))
	}
	if rpm.From != 0 {
		conditions = append(conditions, "time >= :from")
	}
	if rpm.To != 0 {
		conditions = append(conditions, "time < :to")
	}
	if scoped && rpm.DeviceScope != nil {
		conditions = append(conditions, fmt.Sprintf("%s = ANY(:scope_ids)", groupCol))
	}
	where := strings.Join(conditions, " AND ")

	q := fmt.Sprintf(`SELECT %s AS id, MAX(time) AS last_seen, COUNT(*) AS message_count
		FROM %s WHERE %s GROUP BY %s
		ORDER BY last_seen DESC, id ASC
		LIMIT :limit OFFSET :offset;`, groupCol, defTable, where, groupCol)

	totalQ := fmt.Sprintf(`SELECT COUNT(*) FROM (SELECT %s FROM %s WHERE %s GROUP BY %s) AS sub;`, groupCol, defTable, where, groupCol)

	params := map[string]any{
		messageFieldChannel: chanID,
		"filter_val":        filterVal,
		"from":              rpm.From,
		"to":                rpm.To,
		"scope_ids":         rpm.DeviceScope.Devices(),
		"limit":             rpm.Limit,
		"offset":            rpm.Offset,
	}

	page := readers.DeviceStatsPage{PageMetadata: rpm, Stats: []readers.DeviceStat{}}

	rows, err := tr.db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return page, nil
		}
		return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	for rows.Next() {
		var stat deviceStatRow
		if err := rows.StructScan(&stat); err != nil {
			return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		page.Stats = append(page.Stats, readers.DeviceStat{ID: stat.ID, LastSeen: stat.LastSeen, MessageCount: stat.MessageCount})
	}

	totalRows, err := tr.db.NamedQuery(totalQ, params)
	if err != nil {
		if pgErr, ok := pgError(err); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return page, nil
		}
		return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer totalRows.Close()

	if totalRows.Next() {
		if err := totalRows.Scan(&page.Total); err != nil {
			return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
	}

	return page, nil
}

type deviceStatRow struct {
	ID           string  `db:"id"`
	LastSeen     float64 `db:"last_seen"`
	MessageCount uint64  `db:"message_count"`
}

func pgError(err error) (*pgconn.PgError, bool) {
	if preErr, ok := err.(*pgconn.PrepareError); ok {
		err = preErr.Unwrap()
	}
	pgErr, ok := err.(*pgconn.PgError)
	return pgErr, ok
}

// isLegacyJSONDeviceFilter recognises a query against a pre-MG-06 JSON table
// that has no device_id column yet. It has to catch the column both when the
// caller filtered on it explicitly (DeviceIDs) and when an authorization
// scope referenced it implicitly (DeviceScope, set whenever the caller is a
// non-admin domain user regardless of whether they also passed a filter) —
// otherwise a scoped, filter-less request against such a table surfaces as a
// 500 instead of an empty page. The message is matched quoted, the way
// Postgres reports an undefined column, so an unrelated column whose name
// merely contains "device_id" as a substring is not mistaken for this case.
func isLegacyJSONDeviceFilter(format string, rpm readers.PageMetadata, pgErr *pgconn.PgError) bool {
	return format != defTable &&
		(len(rpm.DeviceIDs) > 0 || rpm.DeviceScope != nil) &&
		strings.Contains(pgErr.Message, `"`+messageFieldDeviceID+`"`)
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
			// OR, not AND: a device is named by publisher when it publishes for
			// itself and by device_id when a gateway relays for it, and one row
			// carries only one of the two. The publisher leg is additionally
			// guarded to self-published rows (device_id = '') so that a grant on
			// a gateway client does not also admit every device that gateway
			// has ever relayed for — those rows are named by device_id, not by
			// this gateway's own publisher identity, and must be authorized
			// through the device_id leg instead.
			condition = fmt.Sprintf(`%s AND ((%s = '' AND %s = ANY(:%s)) OR %s = ANY(:%s))`,
				condition, messageFieldDeviceID, messageFieldPublisher, scopeParamPublishers, messageFieldDeviceID, scopeParamDeviceIDs)
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

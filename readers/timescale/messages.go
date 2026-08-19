// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import (
	"encoding/json"
	"fmt"
	"strings"

	api "github.com/absmach/magistrala/api/http"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx" // required for DB access
)

// Table for SenML messages.
const (
	defTable       = "messages"
	orderByTime    = "time"
	orderByCreated = "created"
)

var _ readers.MessageRepository = (*timescaleRepository)(nil)

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

type timescaleRepository struct {
	db *sqlx.DB
}

// New returns new TimescaleSQL writer.
func New(db *sqlx.DB) readers.MessageRepository {
	return &timescaleRepository{
		db: db,
	}
}

func (tr timescaleRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defTable

	if rpm.Format != "" && rpm.Format != defTable {
		format = rpm.Format
	}

	isSenml := (format == defTable)

	// If aggregation is provided, add time_bucket and aggregation to the query
	const timeDivisor = 1000000000
	isAggregated := isSenml && rpm.Aggregation != "" && rpm.Interval != ""

	if rpm.Order == "" {
		switch {
		case isSenml:
			rpm.Order = orderByTime
		default:
			rpm.Order = orderByCreated
		}
	}

	orderClause := applyOrdering(rpm, isAggregated, isSenml)

	pgData := ""
	if rpm.Limit != 0 {
		pgData = "LIMIT :limit"
	}
	if rpm.Offset != 0 {
		if pgData != "" {
			pgData += " "
		}
		pgData += "OFFSET :offset"
	}

	where := fmtCondition(rpm)

	var q string
	totalQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, format, where)

	if isAggregated {
		q = fmt.Sprintf(`
			SELECT
				EXTRACT(epoch FROM time_bucket('%s', to_timestamp(time/%d))) *%d AS time,
				%s(value) AS value,
				FIRST(publisher, time) AS publisher,
				%s,
				FIRST(protocol, time) AS protocol,
				FIRST(subtopic, time) AS subtopic,
				FIRST(name,time) AS name,
				FIRST(unit, time) AS unit
			FROM
				%s
			WHERE
				%s
			GROUP BY 1
			%s
			%s;
			`,
			rpm.Interval, timeDivisor, timeDivisor, rpm.Aggregation, aggregateDeviceIDProjection(rpm), format, where, orderClause, pgData)

		totalQuery = fmt.Sprintf(`SELECT COUNT(*) FROM (SELECT EXTRACT(epoch FROM time_bucket('%s', to_timestamp(time/%d))) AS time, %s(value) AS value FROM %s WHERE %s GROUP BY 1) AS subquery;`, rpm.Interval, timeDivisor, rpm.Aggregation, format, where)
	} else {
		q = fmt.Sprintf(`SELECT * FROM %s WHERE %s %s %s;`, format, where, orderClause, pgData)
	}

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

	rows, err = tr.db.NamedQuery(totalQuery, params)
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
func (tr timescaleRepository) ListGatewayDevices(chanID, publisherID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	return tr.deviceStats(chanID, messageFieldPublisher, publisherID, messageFieldDeviceID, true, rpm)
}

// ListDeviceGateways implements readers.MessageRepository.
func (tr timescaleRepository) ListDeviceGateways(chanID, deviceID string, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
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
//
// The WHERE clause leads with channel then filterCol, matching
// idx_channel_publisher_device_id_time (gateway->devices) and the leading
// columns of MG-06's idx_channel_device_id_name_time (device->gateways), so
// neither direction falls back to a full partition scan.
func (tr timescaleRepository) deviceStats(chanID, filterCol, filterVal, groupCol string, scoped bool, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	conditions := []string{
		fmt.Sprintf("%s = :channel", messageFieldChannel),
		fmt.Sprintf("%s = :filter_val", filterCol),
	}
	if groupCol == messageFieldDeviceID {
		// A publisher's direct (non-relayed) messages carry no device_id at
		// all; grouping without this would surface a spurious "" roster
		// entry for them. publisher, by contrast, is a required column that
		// is never empty, so this guard only applies on this side.
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

func fmtCondition(rpm readers.PageMetadata) string {
	// Indexed columns conditions based on indices order.
	chCondition := " channel = :channel "

	var query map[string]any
	meta, err := json.Marshal(rpm)
	if err != nil {
		return chCondition
	}
	if err := json.Unmarshal(meta, &query); err != nil {
		return chCondition
	}

	conditions := []string{chCondition}

	if _, ok := query[messageFieldSubtopic]; ok {
		conditions = append(conditions, " subtopic = :subtopic ")
	}

	if _, ok := query[messageFieldPublishers]; ok {
		conditions = append(conditions, " publisher = ANY(:publishers) ")
	} else if _, ok := query[messageFieldPublisher]; ok {
		conditions = append(conditions, " publisher = :publisher ")
	}

	// The authorization scope is OR, not AND: a device is named by publisher when
	// it publishes for itself and by device_id when a gateway relays for it, and
	// one row carries only one of the two. The publisher leg is additionally
	// guarded to self-published rows (device_id = '') so that a grant on a
	// gateway client does not also admit every device that gateway has ever
	// relayed for — those rows are named by device_id, not by this gateway's
	// own publisher identity, and must be authorized through the device_id leg
	// instead.
	if _, ok := query[messageFieldScope]; ok {
		conditions = append(conditions, " ( (device_id = '' AND publisher = ANY(:scope_publishers)) OR device_id = ANY(:scope_device_ids) ) ")
	}

	// Ordered to match idx_channel_device_id_name_time, which sits between the
	// publisher and name indexes.
	if _, ok := query[messageFieldDeviceIDs]; ok {
		conditions = append(conditions, " device_id = ANY(:device_ids) ")
	}

	if _, ok := query[messageFieldName]; ok {
		conditions = append(conditions, " name = :name ")
	}

	if _, ok := query["from"]; ok {
		conditions = append(conditions, " time >= :from ")
	}

	if _, ok := query["to"]; ok {
		conditions = append(conditions, " time < :to ")
	}

	// Non Indexed columns conditions added after indexed columns conditions order.
	if _, ok := query[messageFieldProtocol]; ok {
		conditions = append(conditions, " protocol = :protocol ")
	}

	for name := range query {
		switch name {
		case "v":
			comparator := readers.ParseValueComparator(query)
			conditions = append(conditions, fmt.Sprintf(" value %s :value ", comparator))
		case "vb":
			conditions = append(conditions, "bool_value = :bool_value")
		case "vs":
			comparator := readers.ParseValueComparator(query)
			switch comparator {
			case "=":
				conditions = append(conditions, " string_value = :string_value ")
			case ">":
				conditions = append(conditions, " string_value LIKE '%%' || :string_value || '%%' AND string_value <> :string_value ")
			case ">=":
				conditions = append(conditions, " string_value LIKE '%%' || :string_value || '%%' ")
			case "<=":
				conditions = append(conditions, " :string_value LIKE '%%' || string_value || '%%' ")
			case "<":
				conditions = append(conditions, " :string_value LIKE '%%' || string_value || '%%' AND string_value <> :string_value ")
			}
		case "vd":
			comparator := readers.ParseValueComparator(query)
			conditions = append(conditions, fmt.Sprintf(" data_value %s :data_value ", comparator))
		}
	}

	return strings.Join(conditions, " AND ")
}

func aggregateDeviceIDProjection(rpm readers.PageMetadata) string {
	if len(rpm.DeviceIDs) == 1 {
		return "FIRST(device_id, time) AS device_id"
	}
	return "'' AS device_id"
}

type senmlMessage struct {
	ID string `db:"id"`
	senml.Message
}

type jsonMessage struct {
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

func applyOrdering(pm readers.PageMetadata, isAggregated bool, isSenml bool) string {
	timeCol := orderByTime
	if !isSenml {
		timeCol = orderByCreated
	}

	dir := pm.Dir
	if dir != api.AscDir && dir != api.DescDir {
		dir = api.DescDir
	}

	aggCols := map[string]bool{
		orderByTime:           true,
		messageFieldValue:     true,
		"sum":                 true,
		messageFieldPublisher: true,
		messageFieldProtocol:  true,
		messageFieldSubtopic:  true,
		messageFieldName:      true,
		"unit":                true,
	}

	senmlCols := map[string]bool{
		orderByTime:           true,
		messageFieldValue:     true,
		"bool_value":          true,
		"string_value":        true,
		"data_value":          true,
		messageFieldPublisher: true,
		messageFieldName:      true,
		messageFieldProtocol:  true,
		messageFieldChannel:   true,
		messageFieldSubtopic:  true,
		"unit":                true,
	}

	jsonCols := map[string]bool{
		orderByCreated: true, messageFieldPublisher: true, messageFieldProtocol: true,
		messageFieldChannel: true, messageFieldSubtopic: true,
	}

	if isAggregated {
		col := pm.Order
		if !aggCols[col] {
			col = orderByTime
		}
		if col == orderByTime {
			return fmt.Sprintf("ORDER BY time %s", dir)
		}
		return fmt.Sprintf("ORDER BY %s %s, time %s", col, dir, dir)
	}

	col := pm.Order
	switch {
	case isSenml:
		if !senmlCols[col] {
			col = orderByTime
		}
	case !isSenml:
		if !jsonCols[col] {
			col = orderByCreated
		}
	}

	secondary := fmt.Sprintf("%s DESC", timeCol)

	if col == timeCol {
		return fmt.Sprintf("ORDER BY %s %s", col, dir)
	}
	return fmt.Sprintf("ORDER BY %s %s, %s", col, dir, secondary)
}

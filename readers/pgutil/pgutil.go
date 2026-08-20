// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package pgutil holds the message-reader query building blocks shared by
// readers/postgres and readers/timescale. Both backends store messages in a
// `messages` table over the same pgx/Postgres wire protocol — Timescale is a
// Postgres extension, not a different database — so the parts of message
// reading that only depend on that shared schema and driver, not on either
// backend's own time-bucketing or ordering behaviour, belong here once
// instead of twice.
package pgutil

import (
	"fmt"
	"strings"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/readers"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// DeviceStatRow is one row of the MG-15 observed-device aggregation as
// scanned straight off the query: a distinct id (a device serial or a
// gateway's publisher id, depending on direction), its last-seen time, and
// how many messages it produced.
type DeviceStatRow struct {
	ID           string  `db:"id"`
	LastSeen     float64 `db:"last_seen"`
	MessageCount uint64  `db:"message_count"`
}

// PgError unwraps err to a *pgconn.PgError if it is one, looking through the
// wrapping pgx adds around a failed prepare.
func PgError(err error) (*pgconn.PgError, bool) {
	if preErr, ok := err.(*pgconn.PrepareError); ok {
		err = preErr.Unwrap()
	}
	pgErr, ok := err.(*pgconn.PgError)
	return pgErr, ok
}

// IsLegacyJSONDeviceFilter recognises a query against a pre-MG-06 JSON table
// that has no device_id column yet. It has to catch the column both when the
// caller filtered on it explicitly (rpm.DeviceIDs) and when an authorization
// scope referenced it implicitly (rpm.DeviceScope, set whenever the caller
// is a non-admin domain user regardless of whether they also passed a
// filter) — otherwise a scoped, filter-less request against such a table
// surfaces as a 500 instead of an empty page. The message is matched
// quoted, the way Postgres reports an undefined column, so an unrelated
// column whose name merely contains "device_id" as a substring is not
// mistaken for this case.
func IsLegacyJSONDeviceFilter(format, defTable string, rpm readers.PageMetadata, pgErr *pgconn.PgError) bool {
	return format != defTable &&
		(len(rpm.DeviceIDs) > 0 || rpm.DeviceScope != nil) &&
		strings.Contains(pgErr.Message, `"device_id"`)
}

// EmptyPage returns a zero-message page carrying rpm's paging metadata — the
// answer to a query against a table or column that does not exist yet.
func EmptyPage(rpm readers.PageMetadata) readers.MessagesPage {
	return readers.MessagesPage{
		PageMetadata: rpm,
		Messages:     []readers.Message{},
	}
}

// DeviceStats implements both directions of the MG-15 observed-device
// aggregation: distinct values of groupCol, for rows on channel chanID with
// filterCol = filterVal, each with its last-seen time and message count.
//
// scoped applies rpm.DeviceScope to narrow groupCol itself. It is only ever
// true for the gateway->devices direction — see the DeviceScope comment on
// ListDeviceGateways in readers/messages.go for why the inverse direction
// does not narrow by scope at all.
//
// defTable, filterCol and groupCol are threaded through as parameters
// rather than assumed, since the two backends this is shared between name
// their own supporting indexes differently even though both group the same
// channel/device_id/publisher/time columns.
func DeviceStats(db *sqlx.DB, defTable, chanID, filterCol, filterVal, groupCol string, scoped bool, rpm readers.PageMetadata) (readers.DeviceStatsPage, error) {
	conditions := []string{
		"channel = :channel",
		fmt.Sprintf("%s = :filter_val", filterCol),
	}
	if groupCol == "device_id" {
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
		"channel":    chanID,
		"filter_val": filterVal,
		"from":       rpm.From,
		"to":         rpm.To,
		"scope_ids":  rpm.DeviceScope.Devices(),
		"limit":      rpm.Limit,
		"offset":     rpm.Offset,
	}

	page := readers.DeviceStatsPage{PageMetadata: rpm, Stats: []readers.DeviceStat{}}

	rows, err := db.NamedQuery(q, params)
	if err != nil {
		if pgErr, ok := PgError(err); ok && pgErr.Code == pgerrcode.UndefinedTable {
			return page, nil
		}
		return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	defer rows.Close()

	for rows.Next() {
		var stat DeviceStatRow
		if err := rows.StructScan(&stat); err != nil {
			return readers.DeviceStatsPage{}, errors.Wrap(readers.ErrReadMessages, err)
		}
		page.Stats = append(page.Stats, readers.DeviceStat{ID: stat.ID, LastSeen: stat.LastSeen, MessageCount: stat.MessageCount})
	}

	totalRows, err := db.NamedQuery(totalQ, params)
	if err != nil {
		if pgErr, ok := PgError(err); ok && pgErr.Code == pgerrcode.UndefinedTable {
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

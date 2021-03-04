// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/opentracing/opentracing-go"
)

var _ Database = (*database)(nil)

type database struct {
	db *sqlx.DB
}

// Database provides a database interface
type Database interface {
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)
	QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row
	QueryxContext(context.Context, string, ...interface{}) (*sqlx.Rows, error)
	NamedQueryContext(context.Context, string, interface{}) (*sqlx.Rows, error)
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// NewDatabase creates a ThingDatabase instance
func NewDatabase(db *sqlx.DB) Database {
	return &database{
		db: db,
	}
}

func (d database) NamedQueryContext(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error) {
	addSpanTags(ctx, query)
	return d.db.NamedQueryContext(ctx, query, args)
}

func (d database) NamedExecContext(ctx context.Context, query string, args interface{}) (sql.Result, error) {
	addSpanTags(ctx, query)
	return d.db.NamedExecContext(ctx, query, args)
}

func (d database) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	addSpanTags(ctx, query)
	return d.db.QueryRowxContext(ctx, query, args...)
}

func (d database) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	addSpanTags(ctx, query)
	return d.db.QueryxContext(ctx, query, args...)
}

func (d database) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag("span.kind", "client")
		span.SetTag("peer.service", "postgres")
		span.SetTag("db.type", "sql")
	}
	return d.db.BeginTxx(ctx, opts)
}

func addSpanTags(ctx context.Context, query string) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag("sql.statement", query)
		span.SetTag("span.kind", "client")
		span.SetTag("peer.service", "postgres")
		span.SetTag("db.type", "sql")
	}
}

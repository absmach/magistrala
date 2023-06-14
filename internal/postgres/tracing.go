// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ Database = (*database)(nil)

type database struct {
	db     *sqlx.DB
	tracer trace.Tracer
}

// Database provides a database interface.
type Database interface {
	// NamedQueryContext executes a named query against the database and returns
	NamedQueryContext(context.Context, string, interface{}) (*sqlx.Rows, error)

	// NamedExecContext executes a named query against the database and returns
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)

	// QueryRowxContext queries the database and returns an *sqlx.Row
	QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row

	// QueryxContext queries the database and returns an *sqlx.Rows
	QueryxContext(context.Context, string, ...interface{}) (*sqlx.Rows, error)

	// ExecContext executes a query without returning any rows
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)

	// BeginTxx begins a transaction and returns an *sqlx.Tx
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// NewDatabase creates a Clients'Database instance.
func NewDatabase(db *sqlx.DB, tracer trace.Tracer) Database {
	return &database{
		db:     db,
		tracer: tracer,
	}
}

func (d database) NamedQueryContext(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error) {
	ctx, span := d.addSpanTags(ctx, "NamedQueryContext", query)
	defer span.End()
	return d.db.NamedQueryContext(ctx, query, args)
}

func (d database) NamedExecContext(ctx context.Context, query string, args interface{}) (sql.Result, error) {
	ctx, span := d.addSpanTags(ctx, "NamedExecContext", query)
	defer span.End()
	return d.db.NamedExecContext(ctx, query, args)
}

func (d database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := d.addSpanTags(ctx, "ExecContext", query)
	defer span.End()
	return d.db.ExecContext(ctx, query, args...)
}

func (d database) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ctx, span := d.addSpanTags(ctx, "QueryRowxContext", query)
	defer span.End()
	return d.db.QueryRowxContext(ctx, query, args...)
}

func (d database) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, span := d.addSpanTags(ctx, "QueryxContext", query)
	defer span.End()
	return d.db.QueryxContext(ctx, query, args...)
}

func (d database) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	ctx, span := d.addSpanTags(ctx, "sql_beginTxx", "")
	defer span.End()
	return d.db.BeginTxx(ctx, opts)
}

func (d database) addSpanTags(ctx context.Context, method, query string) (context.Context, trace.Span) {
	ctx, span := d.tracer.Start(ctx,
		fmt.Sprintf("sql_%s", method),
		trace.WithAttributes(
			attribute.String("sql.statement", query),
			attribute.String("span.kind", "client"),
			attribute.String("peer.service", "postgres"),
			attribute.String("db.type", "sql"),
		),
	)
	return ctx, span
}

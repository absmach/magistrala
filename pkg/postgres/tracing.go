// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ Database = (*database)(nil)

type database struct {
	Config
	db     *sqlx.DB
	tracer trace.Tracer
}

// Database provides a database interface.
type Database interface {
	// NamedQueryContext executes a named query against the database and returns
	NamedQueryContext(context.Context, string, interface{}) (*sqlx.Rows, error)

	// NamedExecContext executes a named query against the database and returns
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)

	// QueryRowxContext queries the database and returns an *sqlx.Row.
	QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row

	// QueryxContext queries the database and returns an *sqlx.Rows and an error.
	QueryxContext(context.Context, string, ...interface{}) (*sqlx.Rows, error)

	// QueryContext queries the database and returns an *sql.Rows and an error.
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// ExecContext executes a query without returning any rows.
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)

	// BeginTxx begins a transaction and returns an *sqlx.Tx.
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

// NewDatabase creates a Clients'Database instance.
func NewDatabase(db *sqlx.DB, config Config, tracer trace.Tracer) Database {
	database := &database{
		Config: config,
		db:     db,
		tracer: tracer,
	}

	return database
}

func (d *database) NamedQueryContext(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error) {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()

	return d.db.NamedQueryContext(ctx, query, args)
}

func (d *database) NamedExecContext(ctx context.Context, query string, args interface{}) (sql.Result, error) {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()

	return d.db.NamedExecContext(ctx, query, args)
}

func (d *database) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()

	return d.db.ExecContext(ctx, query, args...)
}

func (d *database) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()

	return d.db.QueryRowxContext(ctx, query, args...)
}

func (d *database) QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()

	return d.db.QueryxContext(ctx, query, args...)
}

func (d database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, span := d.addSpanTags(ctx, query)
	defer span.End()
	return d.db.QueryContext(ctx, query, args...)
}

func (d database) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	ctx, span := d.addSpanTags(ctx, "BeginTxx")
	defer span.End()

	return d.db.BeginTxx(ctx, opts)
}

func (d *database) addSpanTags(ctx context.Context, query string) (context.Context, trace.Span) {
	operation := strings.Replace(strings.Split(query, " ")[0], "(", "", 1)

	ctx, span := d.tracer.Start(ctx,
		fmt.Sprintf("%s %s", operation, d.Name),
		trace.WithAttributes(
			// Related to the database instance (informational)
			attribute.String("db.system", "postgresql"),
			attribute.String("db.user", d.User),
			attribute.String("network.transport", "tcp"),
			attribute.String("network.type", "ipv4"),
			attribute.String("server.address", d.Host),
			attribute.String("server.port", d.Port),
			attribute.String("db.name", d.Name),
			attribute.String("db.statement", query),

			// General Span tags
			attribute.String("span.kind", "client"),
		),
	)

	return ctx, span
}

// Copyright (c) Abstract Machines
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
	NamedExecContext(context.Context, string, any) (sql.Result, error)
	QueryRowxContext(context.Context, string, ...any) *sqlx.Row
	NamedQueryContext(context.Context, string, any) (*sqlx.Rows, error)
	GetContext(context.Context, any, string, ...any) error
}

// NewDatabase creates a SubscriptionsDatabase instance.
func NewDatabase(db *sqlx.DB, tracer trace.Tracer) Database {
	return &database{
		db:     db,
		tracer: tracer,
	}
}

func (dm database) NamedExecContext(ctx context.Context, query string, args any) (sql.Result, error) {
	ctx, span := dm.addSpanTags(ctx, "NamedExecContext", query)
	defer span.End()
	return dm.db.NamedExecContext(ctx, query, args)
}

func (dm database) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	ctx, span := dm.addSpanTags(ctx, "QueryRowxContext", query)
	defer span.End()
	return dm.db.QueryRowxContext(ctx, query, args...)
}

func (dm database) NamedQueryContext(ctx context.Context, query string, args any) (*sqlx.Rows, error) {
	ctx, span := dm.addSpanTags(ctx, "NamedQueryContext", query)
	defer span.End()
	return dm.db.NamedQueryContext(ctx, query, args)
}

func (dm database) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	ctx, span := dm.addSpanTags(ctx, "GetContext", query)
	defer span.End()
	return dm.db.GetContext(ctx, dest, query, args...)
}

func (dm database) addSpanTags(ctx context.Context, method, query string) (context.Context, trace.Span) {
	ctx, span := dm.tracer.Start(ctx,
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

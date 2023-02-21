package sqlxt

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go"
)

var (
	ErrRollback           = errors.New("failed to rollback transaction")
	ErrCommit             = errors.New("failed to commit transaction")
	ErrResultRowsAffected = errors.New("failed to get result rows affected")
)
var _ Database = (*database)(nil)

type database struct {
	db *sqlx.DB
}

// Database provides a database interface
type Database interface {
	NamedCUDContext(ctx context.Context, query string, args interface{}) (int64, error, error)
	NamedTotalQueryContext(ctx context.Context, query string, params interface{}) (uint64, error)
	NamedExecContext(context.Context, string, interface{}) (sql.Result, error)
	QueryRowxContext(context.Context, string, ...interface{}) *sqlx.Row
	NamedQueryContext(context.Context, string, interface{}) (*sqlx.Rows, error)
	GetContext(context.Context, interface{}, string, ...interface{}) error
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

// NewDatabase creates a ThingDatabase instance
func NewDatabase(db *sqlx.DB) Database {
	return &database{
		db: db,
	}
}

func (dm database) NamedCUDContext(ctx context.Context, query string, args interface{}) (int64, error, error) {
	tx, err := dm.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err, nil
	}
	result, err := tx.NamedExecContext(ctx, query, args)
	if err != nil {
		errRoll := tx.Rollback()
		if errRoll != nil {
			return 0, err, errors.Wrap(ErrRollback, errRoll)
		}
		return 0, err, nil
	}

	if err := tx.Commit(); err != nil {
		return 0, nil, errors.Wrap(ErrCommit, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return count, nil, errors.Wrap(ErrResultRowsAffected, err)
	}

	return count, nil, nil
}

func (dm database) NamedTotalQueryContext(ctx context.Context, query string, params interface{}) (uint64, error) {
	rows, err := dm.NamedQueryContext(ctx, query, params)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	total := uint64(0)
	if rows.Next() {
		if err := rows.Scan(&total); err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (dm database) NamedExecContext(ctx context.Context, query string, args interface{}) (sql.Result, error) {
	addSpanTags(ctx, query)
	return dm.db.NamedExecContext(ctx, query, args)
}

func (dm database) QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	addSpanTags(ctx, query)
	return dm.db.QueryRowxContext(ctx, query, args...)
}

func (dm database) NamedQueryContext(ctx context.Context, query string, args interface{}) (*sqlx.Rows, error) {
	addSpanTags(ctx, query)
	return dm.db.NamedQueryContext(ctx, query, args)
}

func (dm database) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	addSpanTags(ctx, query)
	return dm.db.GetContext(ctx, dest, query, args...)
}

func (dm database) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span.SetTag("span.kind", "client")
		span.SetTag("peer.service", "postgres")
		span.SetTag("db.type", "sql")
	}
	return dm.db.BeginTxx(ctx, opts)
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

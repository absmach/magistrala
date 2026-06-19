// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx stdlib driver
	"github.com/jmoiron/sqlx"
)

func openDB(ctx context.Context, name, dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", name, err)
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping %s: %w", name, err)
	}
	return db, nil
}

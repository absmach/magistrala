// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "subscriptions_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS subscriptions (
                        id          VARCHAR(254) PRIMARY KEY,
                        owner_id    VARCHAR(254) NOT NULL,
                        contact     VARCHAR(254),
                        topic       TEXT,
                        UNIQUE(topic, contact)
                    )`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS subscriptions",
				},
			},
		},
	}
}

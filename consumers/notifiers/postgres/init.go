// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of the Notifiers service.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding
// "subscriptions_10" would sort before "subscriptions_2". Keep the same width
// when adding migrations, and never renumber an id that has already shipped.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "subscriptions_0001",
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

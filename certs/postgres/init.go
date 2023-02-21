// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of Certs service
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS certs (
						thing_id     TEXT NOT NULL,
						owner_id     TEXT NOT NULL,
						expire       TIMESTAMPTZ NOT NULL,
						serial       TEXT NOT NULL,
						PRIMARY KEY  (thing_id, owner_id, serial)
					);`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS certs;",
				},
			},

			{
				Id: "certs_2",
				Up: []string{
					`					
					ALTER TABLE certs DROP CONSTRAINT certs_pkey;
					ALTER TABLE certs ADD COLUMN id UUID NOT NULL;
					ALTER TABLE certs ADD COLUMN name VARCHAR(254) NOT NULL;
					ALTER TABLE certs ADD COLUMN certificate TEXT NOT NULL;
					ALTER TABLE certs ADD COLUMN private_key TEXT NOT NULL;
					ALTER TABLE certs ADD COLUMN ca_chain TEXT NOT NULL;
					ALTER TABLE certs ADD COLUMN issuing_ca TEXT NOT NULL;
					ALTER TABLE certs ADD COLUMN ttl VARCHAR(254) NOT NULL;
					ALTER TABLE certs ADD COLUMN revocation TIMESTAMPTZ NULL;
					ALTER TABLE certs ADD PRIMARY KEY (name, thing_id, owner_id);
					`,
				},
				Down: []string{
					`
					ALTER TABLE certs DROP CONSTRAINT certs_pkey;
					ALTER TABLE certs DROP COLUMN id data_type;
					ALTER TABLE certs DROP COLUMN name data_type;
					ALTER TABLE certs DROP COLUMN certificate data_type;
					ALTER TABLE certs DROP COLUMN private_key data_type;
					ALTER TABLE certs DROP COLUMN ca_chain data_type;
					ALTER TABLE certs DROP COLUMN issuing_ca data_type;
					ALTER TABLE certs DROP COLUMN ttl data_type;
					ALTER TABLE certs DROP COLUMN revocation data_type;
					ALTER TABLE certs ADD PRIMARY KEY (thing_id, owner_id, serial);
					`,
				},
			},
		},
	}
}

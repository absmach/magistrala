// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// dbConn is one source/target Postgres connection target.
type dbConn struct {
	Host string
	Port string
	User string
	Pass string
	Name string
	SSL  string
}

func (d dbConn) DSN() string {
	ssl := d.SSL
	if ssl == "" {
		ssl = "disable"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Pass, d.Name, ssl)
}

// config holds every source DB plus the Atom target DSN.
type config struct {
	Domains  dbConn
	Users    dbConn
	Clients  dbConn
	Channels dbConn
	Groups   dbConn
	Auth     dbConn
	RE       dbConn // rules engine
	Reports  dbConn

	AtomDSN        string
	UnmappedAction string

	AtomKeyEncryptionKey   []byte
	AtomKeyEncryptionKeyID string
}

// loadConfig reads docker/.env for MG_*_DB_* keys. When fromHost is true the
// service-name hosts are rewritten to 127.0.0.1 with the mapped host port (the
// caller is then responsible for exposing those ports in compose).
func loadConfig(envPath, atomDSN string, fromHost bool) (config, error) {
	env, err := parseEnvFile(envPath)
	if err != nil {
		return config{}, err
	}

	mk := func(prefix string) dbConn {
		c := dbConn{
			Host: env[prefix+"_DB_HOST"],
			Port: orDef(env[prefix+"_DB_PORT"], "5432"),
			User: orDef(env[prefix+"_DB_USER"], "magistrala"),
			Pass: orDef(env[prefix+"_DB_PASS"], "magistrala"),
			Name: env[prefix+"_DB_NAME"],
			SSL:  orDef(env[prefix+"_DB_SSL_MODE"], "disable"),
		}
		if fromHost {
			c.Host = "127.0.0.1"
		}
		return c
	}

	cfg := config{
		Domains:  mk("MG_DOMAINS"),
		Users:    mk("MG_USERS"),
		Clients:  mk("MG_CLIENTS"),
		Channels: mk("MG_CHANNELS"),
		Groups:   mk("MG_GROUPS"),
		Auth:     mk("MG_AUTH"),
		RE:       mk("MG_RE"),
		Reports:  mk("MG_REPORTS"),
		AtomDSN:  atomDSN,
	}
	if key := strings.TrimSpace(firstNonEmpty(env["ATOM_KEY_ENCRYPTION_KEY"], os.Getenv("ATOM_KEY_ENCRYPTION_KEY"))); key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return config{}, fmt.Errorf("ATOM_KEY_ENCRYPTION_KEY must be base64 encoded: %w", err)
		}
		if len(decoded) != 32 {
			return config{}, fmt.Errorf("ATOM_KEY_ENCRYPTION_KEY must decode to exactly 32 bytes")
		}
		cfg.AtomKeyEncryptionKey = decoded
	}
	cfg.AtomKeyEncryptionKeyID = orDef(strings.TrimSpace(firstNonEmpty(env["ATOM_KEY_ENCRYPTION_KEY_ID"], os.Getenv("ATOM_KEY_ENCRYPTION_KEY_ID"))), "local:v1")

	// Default names if .env omitted them.
	defName := map[*string]string{
		&cfg.Domains.Name: collectionDomains, &cfg.Users.Name: "users",
		&cfg.Clients.Name: "clients", &cfg.Channels.Name: "channels",
		&cfg.Groups.Name: collectionGroups, &cfg.Auth.Name: "auth",
		&cfg.RE.Name: "rules_engine", &cfg.Reports.Name: "reports",
	}
	for p, n := range defName {
		if *p == "" {
			*p = n
		}
	}

	if cfg.AtomDSN == "" {
		// Fall back to Atom compose defaults; override with --atom-dsn or
		// ATOM_DATABASE_URL for real runs.
		cfg.AtomDSN = "host=127.0.0.1 port=5432 user=atom password=atom dbname=atom sslmode=disable"
	}
	return cfg, nil
}

func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out, sc.Err()
}

func orDef(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

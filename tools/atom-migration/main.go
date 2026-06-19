// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Command atom-migration migrates a Magistrala v0.30.0 deployment (per-service
// Postgres databases) into a single Atom IAM Postgres database.
//
// It is offline and idempotent: every write upserts on a preserved/derived
// primary key, so the tool is safe to re-run. Default mode is --dry-run, which
// reads, transforms, validates and reports without writing anything.
//
// See PLAN.md in this directory for the full mapping and runbook.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	var (
		envPath    = flag.String("env", "docker/.env", "path to Magistrala docker/.env (DB hosts/creds)")
		atomDSN    = flag.String("atom-dsn", envOr("ATOM_DATABASE_URL", ""), "Atom Postgres DSN (overrides --env atom block)")
		apply      = flag.Bool("apply", false, "perform the load (default is dry-run: read+validate+report only)")
		reportDir  = flag.String("report-dir", "tools/atom-migration/report", "directory for the JSON+markdown report")
		fromHost   = flag.Bool("from-host", false, "connect to source DBs via localhost mapped ports instead of compose service names")
		unmappedOK = flag.String("unmapped-action", "manage", "fallback Atom action for unmapped Magistrala actions: manage|skip")
	)
	flag.Parse()

	cfg, err := loadConfig(*envPath, *atomDSN, *fromHost)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg.UnmappedAction = *unmappedOK

	mode := "DRY-RUN"
	if *apply {
		mode = "APPLY"
	}
	log.Printf("atom-migration starting (%s)", mode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	m, err := newMigrator(ctx, cfg, *apply)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer m.Close()

	rep := newReport(mode)
	if err := m.Run(ctx, rep); err != nil {
		rep.Errorf("fatal: %v", err)
		_ = rep.Write(*reportDir)
		log.Fatalf("run: %v", err)
	}

	if err := rep.Write(*reportDir); err != nil {
		log.Fatalf("report: %v", err)
	}
	fmt.Print(rep.Summary())
	if rep.HasBlocking() && *apply {
		os.Exit(2)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

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
	os.Exit(runMain())
}

func runMain() int {
	var (
		envPath    = flag.String("env", "docker/.env", "path to Magistrala docker/.env (DB hosts/creds)")
		atomDSN    = flag.String("atom-dsn", envOr("ATOM_DATABASE_URL", ""), "Atom Postgres DSN (overrides --env atom block)")
		apply      = flag.Bool("apply", false, "perform the load (default is dry-run: read+validate+report only)")
		reportDir  = flag.String("report-dir", "tools/atom-migration/report", "directory for the JSON+markdown report")
		fromHost   = flag.Bool("from-host", false, "connect to source DBs via localhost mapped ports instead of compose service names")
		unmappedOK = flag.String("unmapped-action", "manage", "fallback Atom action for unmapped Magistrala actions: manage|skip")
		verify     = flag.Bool("verify", false, "verify a completed migration (reconcile source vs Atom); read-only, no load")
	)
	flag.Parse()

	cfg, err := loadConfig(*envPath, *atomDSN, *fromHost)
	if err != nil {
		log.Printf("config: %v", err)
		return 1
	}
	cfg.UnmappedAction = *unmappedOK

	mode := "DRY-RUN"
	switch {
	case *verify:
		mode = "VERIFY"
	case *apply:
		mode = "APPLY"
	}
	log.Printf("atom-migration starting (%s)", mode)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	m, err := newMigrator(ctx, cfg, *apply)
	if err != nil {
		log.Printf("init: %v", err)
		return 1
	}
	m.reportDir = *reportDir
	defer m.Close()

	rep := newReport(mode)
	run := m.Run
	if *verify {
		run = m.Verify
	}
	if err := run(ctx, rep); err != nil {
		rep.Errorf("fatal: %v", err)
		if writeErr := rep.Write(*reportDir); writeErr != nil {
			log.Printf("report: %v", writeErr)
		}
		log.Printf("%s: %v", mode, err)
		return 1
	}

	if err := rep.Write(*reportDir); err != nil {
		log.Printf("report: %v", err)
		return 1
	}
	fmt.Print(rep.Summary())
	if rep.HasBlocking() && *apply {
		return 2
	}
	return 0
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

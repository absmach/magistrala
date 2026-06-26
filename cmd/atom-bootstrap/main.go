// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the one-shot Magistrala Atom bootstrap command.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/absmach/magistrala/internal/atom"
)

const (
	defaultRetries       = 30
	defaultRetryInterval = 2 * time.Second
	defaultTimeout       = 30 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg := atom.LoadConfig()
	if cfg.URL == "" {
		log.Fatal("ATOM_URL is required")
	}

	client := atom.NewClient(cfg)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "bootstrap-actions":
			runBootstrapActions(client)
			return
		case "provision-tokens":
			runProvisionTokens(client, os.Args[2:])
			return
		default:
			log.Fatalf("unknown command %q", os.Args[1])
		}
	}
	runBootstrapActions(client)
}

func runBootstrapActions(client *atom.Client) {
	retries := envInt("MG_ATOM_BOOTSTRAP_RETRIES", defaultRetries)
	retryInterval := envDuration("MG_ATOM_BOOTSTRAP_RETRY_INTERVAL", defaultRetryInterval)
	timeout := envDuration("MG_ATOM_BOOTSTRAP_TIMEOUT", defaultTimeout)

	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err := atom.BootstrapMagistralaActions(ctx, client)
		cancel()
		if err == nil {
			log.Printf("Magistrala Atom action bootstrap completed")
			return
		}
		lastErr = err
		if attempt < retries {
			log.Printf("Magistrala Atom action bootstrap attempt %d/%d failed: %v; retrying in %s", attempt, retries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	log.Fatalf("Magistrala Atom action bootstrap failed after %d attempts: %v", retries, lastErr)
}

func runProvisionTokens(client *atom.Client, args []string) {
	fs := flag.NewFlagSet("provision-tokens", flag.ExitOnError)
	output := fs.String("output", envString("MG_ATOM_TOKENS_OUTPUT", "docker/.env.tokens"), "path to write generated token env file")
	rotate := fs.String("rotate", "", "rotate one token by name/env var, or all")
	entityID := fs.String("entity-id", envString("ATOM_SERVICE_ENTITY_ID", atom.DefaultServiceEntityID), "Atom service entity ID to receive API keys")
	if err := fs.Parse(args); err != nil {
		log.Fatal(err)
	}

	retries := envInt("MG_ATOM_BOOTSTRAP_RETRIES", defaultRetries)
	retryInterval := envDuration("MG_ATOM_BOOTSTRAP_RETRY_INTERVAL", defaultRetryInterval)
	timeout := envDuration("MG_ATOM_BOOTSTRAP_TIMEOUT", defaultTimeout)

	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		result, err := atom.ProvisionServiceTokens(ctx, client, atom.TokenProvisionOptions{
			OutputPath:      *output,
			ServiceEntityID: *entityID,
			Rotate:          *rotate,
		})
		cancel()
		if err == nil {
			log.Printf("Magistrala Atom token provisioning completed: output=%s preserved=%d created=%d rotated=%d",
				result.OutputPath, len(result.Preserved), len(result.Created), len(result.Rotated))
			return
		}
		lastErr = err
		if attempt < retries {
			log.Printf("Magistrala Atom token provisioning attempt %d/%d failed: %v; retrying in %s", attempt, retries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	log.Fatalf("Magistrala Atom token provisioning failed after %d attempts: %v", retries, lastErr)
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err == nil && value > 0 {
		return value
	}
	seconds, err := strconv.Atoi(raw)
	if err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	fmt.Fprintf(os.Stderr, "invalid %s=%q, using %s\n", key, raw, fallback)
	return fallback
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains the one-shot Magistrala Atom bootstrap command.
package main

import (
	"context"
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

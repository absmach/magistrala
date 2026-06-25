// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// report accumulates per-phase counts, skips, and blocking issues.
type report struct {
	mu sync.Mutex

	Mode      string         `json:"mode"`
	StartedAt time.Time      `json:"started_at"`
	Counts    map[string]int `json:"counts"`   // phase -> rows written/planned
	Skipped   map[string]int `json:"skipped"`  // reason -> count
	Warnings  []string       `json:"warnings"` // non-blocking
	Blocking  []string       `json:"blocking"` // must fix before --apply
	Errors    []string       `json:"errors"`

	// Follow-up lists surfaced for operators (force-reset users, re-issue PATs, etc.)
	Todo map[string][]string `json:"todo"`
}

func newReport(mode string) *report {
	return &report{
		Mode:      mode,
		StartedAt: time.Now().UTC(),
		Counts:    map[string]int{},
		Skipped:   map[string]int{},
		Todo:      map[string][]string{},
	}
}

func (r *report) count(phase string, n int) {
	r.mu.Lock()
	r.Counts[phase] += n
	r.mu.Unlock()
}

func (r *report) skip(reason string) {
	r.mu.Lock()
	r.Skipped[reason]++
	r.mu.Unlock()
}

func (r *report) warnf(format string, a ...any) {
	r.mu.Lock()
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
	r.mu.Unlock()
}

func (r *report) blockf(format string, a ...any) {
	r.mu.Lock()
	r.Blocking = append(r.Blocking, fmt.Sprintf(format, a...))
	r.mu.Unlock()
}

func (r *report) Errorf(format string, a ...any) {
	r.mu.Lock()
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
	r.mu.Unlock()
}

func (r *report) todo(bucket, item string) {
	r.mu.Lock()
	r.Todo[bucket] = append(r.Todo[bucket], item)
	r.mu.Unlock()
}

func (r *report) HasBlocking() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Blocking) > 0 || len(r.Errors) > 0
}

func (r *report) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	stamp := r.StartedAt.Format("20060102-150405")
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "report-"+stamp+".json"), b, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "report-"+stamp+".md"), []byte(r.markdown()), 0o644)
}

func (r *report) Summary() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== atom-migration %s ===\n", r.Mode)
	for _, k := range sortedKeys(r.Counts) {
		fmt.Fprintf(&b, "  %-28s %d\n", k, r.Counts[k])
	}
	if len(r.Skipped) > 0 {
		b.WriteString("  skipped:\n")
		for _, k := range sortedKeys(r.Skipped) {
			fmt.Fprintf(&b, "    %-26s %d\n", k, r.Skipped[k])
		}
	}
	fmt.Fprintf(&b, "  warnings=%d blocking=%d errors=%d\n",
		len(r.Warnings), len(r.Blocking), len(r.Errors))
	for _, x := range r.Blocking {
		fmt.Fprintf(&b, "  BLOCK: %s\n", x)
	}
	for _, x := range r.Errors {
		fmt.Fprintf(&b, "  ERROR: %s\n", x)
	}
	return b.String()
}

func (r *report) markdown() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	fmt.Fprintf(&b, "# atom-migration report (%s)\n\n%s\n\n", r.Mode, r.StartedAt.Format(time.RFC3339))
	b.WriteString("## Counts\n\n| phase | rows |\n|---|---|\n")
	for _, k := range sortedKeys(r.Counts) {
		fmt.Fprintf(&b, "| %s | %d |\n", k, r.Counts[k])
	}
	if len(r.Skipped) > 0 {
		b.WriteString("\n## Skipped\n\n| reason | count |\n|---|---|\n")
		for _, k := range sortedKeys(r.Skipped) {
			fmt.Fprintf(&b, "| %s | %d |\n", k, r.Skipped[k])
		}
	}
	section := func(title string, items []string) {
		if len(items) == 0 {
			return
		}
		fmt.Fprintf(&b, "\n## %s\n\n", title)
		for _, x := range items {
			fmt.Fprintf(&b, "- %s\n", x)
		}
	}
	section("Blocking", r.Blocking)
	section("Errors", r.Errors)
	section("Warnings", r.Warnings)
	for _, bucket := range sortedKeys2(r.Todo) {
		section("TODO: "+bucket, r.Todo[bucket])
	}
	return b.String()
}

func sortedKeys(m map[string]int) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedKeys2(m map[string][]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

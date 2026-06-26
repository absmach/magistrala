// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// uuidNS is the deterministic namespace for derived Atom UUIDs (roles,
// permission blocks) whose Magistrala source id is not a UUID. Using uuidv5
// keeps the migration idempotent across re-runs.
var uuidNS = uuid.MustParse("a70a0000-0000-5000-a000-000000000000")

const (
	statusActive = "active"

	actionCreate    = "create"
	actionDelete    = "delete"
	actionManage    = "manage"
	actionPublish   = "publish"
	actionRead      = "read"
	actionSubscribe = "subscribe"
)

func derivedUUID(parts ...string) string {
	return uuid.NewSHA1(uuidNS, []byte(strings.Join(parts, "|"))).String()
}

// --- status mapping (Magistrala smallint -> Atom enum text) ---

// entityStatus maps to entities.status / object_groups.status (active/inactive/suspended).
func entityStatus(s int16) string {
	switch s {
	case 0:
		return statusActive
	case 1:
		return "inactive"
	default:
		return "suspended"
	}
}

// tenantStatus maps to tenants.status (active/inactive/frozen/deleted).
func tenantStatus(s int16) string {
	switch s {
	case 0:
		return statusActive
	case 1:
		return "inactive"
	case 2:
		return "frozen"
	default:
		return statusActive
	}
}

// --- alias normalization (Atom 004/005 slug rules) ---

var (
	slugRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	uuidRe = regexp.MustCompile(`^([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
)

// normalizeAlias lowercases and validates a candidate alias. Returns ("", false)
// when the value cannot be a valid Atom alias (caller keeps the UUID identity and
// drops the alias, logging it).
func normalizeAlias(raw string) (string, bool) {
	a := strings.ToLower(strings.TrimSpace(raw))
	if a == "" {
		return "", false
	}
	if uuidRe.MatchString(a) {
		return "", false // UUID-shaped aliases collide with id-addressing
	}
	if !slugRe.MatchString(a) {
		return "", false
	}
	return a, true
}

// --- action vocabulary mapping (Magistrala -> Atom) ---

// Atom actions: read create write delete revoke rotate publish subscribe execute
// manage policy.manage role.manage authz.check.
//
// mapAction translates a Magistrala role action string. Returns ("", false) when
// unmapped; caller decides fallback per --unmapped-action.
func mapAction(a string) (string, bool) {
	a = strings.ToLower(strings.TrimSpace(a))
	switch a {
	case actionPublish:
		return actionPublish, true
	case actionSubscribe:
		return actionSubscribe, true
	case actionCreate:
		return actionCreate, true
	case "update":
		return "write", true
	case actionDelete:
		return actionDelete, true
	case "read", "view":
		return actionRead, true
	case "admin", actionManage:
		return actionManage, true
	}
	switch {
	case strings.HasSuffix(a, "_publish"):
		return actionPublish, true
	case strings.HasSuffix(a, "_subscribe"):
		return actionSubscribe, true
	case strings.HasSuffix(a, "_view_role_users"), strings.HasSuffix(a, "_read"),
		strings.HasSuffix(a, "_view"), strings.Contains(a, "_read_"):
		return actionRead, true
	case strings.Contains(a, actionCreate):
		return actionCreate, true
	case strings.HasSuffix(a, "_update"):
		return "write", true
	case strings.Contains(a, "_delete"):
		return actionDelete, true
	case strings.HasSuffix(a, "_manage_role"):
		return actionManage, true
	case strings.HasSuffix(a, "_add_role_users"), strings.HasSuffix(a, "_remove_role_users"),
		strings.Contains(a, "membership"), strings.Contains(a, "_connect"):
		return "policy.manage", true
	case strings.HasSuffix(a, "_share"), strings.HasSuffix(a, "_unshare"):
		return "policy.manage", true
	}
	return "", false
}

// connectionAction maps Magistrala connection type (1=publish, 2=subscribe).
func connectionAction(t int16) (string, bool) {
	switch t {
	case 1:
		return actionPublish, true
	case 2:
		return actionSubscribe, true
	default:
		return "", false
	}
}

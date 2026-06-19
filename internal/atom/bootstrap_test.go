// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBootstrapMagistralaActionsCreatesMissingActionsAndApplicability(t *testing.T) {
	actions := map[string]Capability{
		"read":   {ID: "read-id", Name: "read"},
		"write":  {ID: "write-id", Name: "write"},
		"delete": {ID: "delete-id", Name: "delete"},
		"manage": {ID: "manage-id", Name: "manage"},
	}
	var applicability []map[string]any
	var assignmentRules []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(payload.Query, "query Actions"):
			items := make([]Capability, 0, len(actions))
			for _, action := range actions {
				items = append(items, action)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"actions": map[string]any{"items": items, "total": len(items)},
				},
			})
		case strings.Contains(payload.Query, "query ActionAssignmentRules"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"actionAssignmentRules": map[string]any{"items": []map[string]any{}, "total": 0},
				},
			})
		case strings.Contains(payload.Query, "createActionAssignmentRule"):
			input := payload.Variables["input"].(map[string]any)
			assignmentRules = append(assignmentRules, input)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createActionAssignmentRule": map[string]any{
						"id":          input["actionName"].(string) + "-rule-id",
						"tenant_id":   "",
						"entity_kind": input["entityKind"],
						"action_name": input["actionName"],
						"object_kind": input["objectKind"],
						"object_type": input["objectType"],
						"decision":    input["decision"],
						"is_absolute": input["isAbsolute"],
						"created_at":  "2026-06-18T00:00:00Z",
					},
				},
			})
		case strings.Contains(payload.Query, "createAction"):
			input := payload.Variables["input"].(map[string]any)
			name := input["name"].(string)
			action := Capability{ID: name + "-id", Name: name}
			actions[name] = action
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"createAction": action},
			})
		case strings.Contains(payload.Query, "addActionApplicability"):
			input := payload.Variables["input"].(map[string]any)
			applicability = append(applicability, input)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"addActionApplicability": map[string]any{
						"action_id":   input["actionId"],
						"action_name": "action",
						"object_kind": input["objectKind"],
						"object_type": input["objectType"],
						"description": "",
					},
				},
			})
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	if err := BootstrapMagistralaActions(context.Background(), client); err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	for _, name := range []string{"read", "write", "delete", "manage", "publish", "subscribe", "execute"} {
		if _, ok := actions[name]; !ok {
			t.Fatalf("action %q was not ensured", name)
		}
	}
	if len(applicability) != len(magistralaActionApplicability) {
		t.Fatalf("unexpected applicability count: got %d want %d", len(applicability), len(magistralaActionApplicability))
	}
	assertApplicability(t, applicability, "publish-id", "resource:channel")
	assertApplicability(t, applicability, "execute-id", "resource:rule")
	assertApplicability(t, applicability, "execute-id", "resource:report")
	assertApplicability(t, applicability, "manage-id", "resource:alarm")
	if len(assignmentRules) != len(magistralaActionAssignmentRules) {
		t.Fatalf("unexpected assignment guardrail count: got %d want %d", len(assignmentRules), len(magistralaActionAssignmentRules))
	}
	assertAssignmentRule(t, assignmentRules, "device", "publish", "resource", "resource:channel", "allow")
	assertAssignmentRule(t, assignmentRules, "device", "subscribe", "resource", "resource:channel", "allow")
}

func assertApplicability(t *testing.T, entries []map[string]any, actionID, objectType string) {
	t.Helper()
	for _, entry := range entries {
		if entry["actionId"] == actionID && entry["objectKind"] == "resource" && entry["objectType"] == objectType {
			return
		}
	}
	t.Fatalf("missing applicability action=%s object_type=%s", actionID, objectType)
}

func assertAssignmentRule(t *testing.T, entries []map[string]any, entityKind, actionName, objectKind, objectType, decision string) {
	t.Helper()
	for _, entry := range entries {
		if entry["entityKind"] == entityKind &&
			entry["actionName"] == actionName &&
			entry["objectKind"] == objectKind &&
			entry["objectType"] == objectType &&
			entry["decision"] == decision &&
			entry["isAbsolute"] == false {
			return
		}
	}
	t.Fatalf("missing assignment guardrail entity=%s action=%s object=%s:%s decision=%s", entityKind, actionName, objectKind, objectType, decision)
}

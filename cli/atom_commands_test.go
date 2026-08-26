// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// recordingServer answers every GraphQL call with body and captures the request
// for assertions on the test goroutine, since the handler runs on its own.
func recordingServer(t *testing.T, body string) (*httptest.Server, *graphQLRequest) {
	t.Helper()
	var req graphQLRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server, &req
}

func runRootCmd(t *testing.T, args ...string) error {
	t.Helper()
	cmd := NewRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	return cmd.Execute()
}

func TestChannelSelectionsUseAtomObjectGroupFields(t *testing.T) {
	clearAtomEnv(t)
	server, req := recordingServer(t, `{"data":{"resources":{"total":0,"items":[]}}}`)

	err := runRootCmd(t, "--graphql-url", server.URL, "--token", "test-token", cmdChannels, useList, "workspace-1")
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if strings.Contains(req.Query, "parentGroupId") {
		t.Errorf("query uses stale parentGroupId field: %s", req.Query)
	}
	if !strings.Contains(req.Query, "objectGroupIds") {
		t.Errorf("query does not select objectGroupIds: %s", req.Query)
	}
}

func TestGroupCreateUsesSpecializedAtomMutations(t *testing.T) {
	tests := []struct {
		name         string
		groupType    string
		mutationName string
	}{
		{name: "object group", groupType: groupTypeObject, mutationName: respCreateObjectGroup},
		{name: "principal group", groupType: groupTypePrincipal, mutationName: respCreatePrincipalGroup},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clearAtomEnv(t)
			server, req := recordingServer(t, `{"data":{"`+tc.mutationName+`":{"id":"group-1","name":"Group"}}}`)

			err := runRootCmd(t,
				"--graphql-url", server.URL,
				"--token", "test-token",
				cmdGroups, "create", "workspace-1", "Group",
				"--type", tc.groupType,
			)
			if err != nil {
				t.Fatalf("execute command: %v", err)
			}

			if !strings.Contains(req.Query, tc.mutationName+"(input: $input)") {
				t.Errorf("query does not use %s: %s", tc.mutationName, req.Query)
			}
			if strings.Contains(req.Query, "createGroup(input: $input)") {
				t.Errorf("query still uses generic createGroup: %s", req.Query)
			}
			input, ok := req.Variables[varInput].(map[string]any)
			if !ok {
				t.Fatalf("missing input variables: %#v", req.Variables)
			}
			if _, ok := input["groupType"]; ok {
				t.Errorf("groupType should not be sent to a specialized mutation: %#v", input)
			}
		})
	}
}

func TestGroupCreateRejectsUnsupportedGroupType(t *testing.T) {
	clearAtomEnv(t)
	// No endpoint is configured: the group type has to be rejected before the
	// command reaches the network.
	err := runRootCmd(t, "--token", "test-token", cmdGroups, "create", "workspace-1", "Group", "--type", "invalid")
	if err == nil {
		t.Fatal("expected unsupported group type error")
	}
	if !strings.Contains(err.Error(), "expected object or principal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPasswordChangeUsesVerifiedSelfMutation(t *testing.T) {
	clearAtomEnv(t)
	server, req := recordingServer(t, `{"data":{"`+respChangePassword+`":true}}`)

	err := runRootCmd(t,
		"--graphql-url", server.URL+atomGraphQLPath,
		"--token", "test-token",
		"password", "change", "old-password", "new-password",
	)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if !strings.Contains(req.Query, "changeOwnPassword") {
		t.Errorf("query does not change own password: %s", req.Query)
	}
	if strings.Contains(req.Query, "createPassword") {
		t.Errorf("password change must not create a replacement credential directly: %s", req.Query)
	}
	if req.Variables["currentPassword"] != "old-password" {
		t.Errorf("unexpected current password variable: %#v", req.Variables["currentPassword"])
	}
	if req.Variables["newPassword"] != "new-password" {
		t.Errorf("unexpected new password variable: %#v", req.Variables["newPassword"])
	}
}

func TestWorkspaceCreateSeedsDefaultDeviceTypes(t *testing.T) {
	clearAtomEnv(t)
	createdKeys := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(req.Query, "createTenant"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createTenant": map[string]any{
						"id":     "domain-1",
						"name":   "Domain",
						"route":  "domain",
						"status": "active",
					},
				},
			})
		case strings.Contains(req.Query, "query DeviceTypes("):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"profiles": map[string]any{"total": 0, "items": []any{}},
				},
			})
		case strings.Contains(req.Query, "mutation CreateDeviceType("):
			input, ok := req.Variables[varInput].(map[string]any)
			if !ok {
				t.Fatalf("unexpected profile input: %#v", req.Variables[varInput])
			}
			key, _ := input["key"].(string)
			createdKeys[key] = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createProfile": map[string]any{
						"id":        "profile-" + key,
						"tenant_id": "domain-1",
						"key":       key,
						"name":      input["displayName"],
						"status":    "active",
					},
				},
			})
		case strings.Contains(req.Query, "query DeviceTypeVersions("):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"profileVersions": []any{}},
			})
		case strings.Contains(req.Query, "mutation CreateDeviceTypeVersion("):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"createProfileVersion": map[string]any{
						"id":             "version-1",
						"device_type_id": req.Variables["profileId"],
						"version":        1,
						"json_schema":    req.Variables[varInput].(map[string]any)["jsonSchema"],
						"ui_schema":      req.Variables[varInput].(map[string]any)["uiSchema"],
						"status":         "active",
					},
				},
			})
		default:
			t.Fatalf("unexpected query: %s", req.Query)
		}
	}))
	defer server.Close()

	err := runRootCmd(t,
		"--graphql-url", server.URL+atomGraphQLPath,
		"--token", "test-token",
		cmdWorkspaces, "create", "Workspace",
		"--alias", "workspace",
	)
	if err != nil {
		t.Fatalf("execute command: %v", err)
	}

	for _, key := range []string{"water-meter", "pressure-sensor", "energy-meter", "pump-controller"} {
		if !createdKeys[key] {
			t.Fatalf("default device type %q was not created: %+v", key, createdKeys)
		}
	}
}

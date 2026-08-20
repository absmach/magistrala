// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientAndChannelSelectionsUseAtomObjectGroupFields(t *testing.T) {
	tests := []struct {
		name string
		args []string
		data string
	}{
		{
			name: "clients list",
			args: []string{"--graphql-url", "", "--token", "test-token", "clients", "list", "domain-1"},
			data: `{"data":{"entities":{"total":0,"items":[]}}}`,
		},
		{
			name: "channels list",
			args: []string{"--graphql-url", "", "--token", "test-token", "channels", "list", "domain-1"},
			data: `{"data":{"resources":{"total":0,"items":[]}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req graphQLRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode graphql request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.data))
			}))
			defer server.Close()

			tc.args[1] = server.URL
			cmd := NewRootCmd()
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute command: %v", err)
			}

			if strings.Contains(req.Query, "parentGroupId") {
				t.Fatalf("query uses stale parentGroupId field: %s", req.Query)
			}
			if !strings.Contains(req.Query, "objectGroupIds") {
				t.Fatalf("query does not select objectGroupIds: %s", req.Query)
			}
		})
	}
}

func TestGroupCreateUsesSpecializedAtomMutations(t *testing.T) {
	tests := []struct {
		name         string
		groupType    string
		mutationName string
	}{
		{name: "object group", groupType: "object", mutationName: "createObjectGroup"},
		{name: "principal group", groupType: "principal", mutationName: "createPrincipalGroup"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req graphQLRequest
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode graphql request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"` + tc.mutationName + `":{"id":"group-1","name":"Group"}}}`))
			}))
			defer server.Close()

			cmd := NewRootCmd()
			cmd.SetArgs([]string{
				"--graphql-url", server.URL,
				"--token", "test-token",
				"groups", "create", "domain-1", "Group",
				"--type", tc.groupType,
			})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("execute command: %v", err)
			}

			if !strings.Contains(req.Query, tc.mutationName+"(input: $input)") {
				t.Fatalf("query does not use %s: %s", tc.mutationName, req.Query)
			}
			if strings.Contains(req.Query, "createGroup(input: $input)") {
				t.Fatalf("query still uses generic createGroup: %s", req.Query)
			}
			input, ok := req.Variables["input"].(map[string]any)
			if !ok {
				t.Fatalf("missing input variables: %#v", req.Variables)
			}
			if _, ok := input["groupType"]; ok {
				t.Fatalf("groupType should not be sent to specialized mutation: %#v", input)
			}
		})
	}
}

func TestGroupCreateRejectsUnsupportedGroupType(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--token", "test-token", "groups", "create", "domain-1", "Group", "--type", "invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unsupported group type error")
	}
	if !strings.Contains(err.Error(), "expected object or principal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

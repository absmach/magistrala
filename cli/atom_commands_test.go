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

	err := runRootCmd(t, "--graphql-url", server.URL, "--token", "test-token", cmdChannels, useList, "domain-1")
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
				cmdGroups, "create", "domain-1", "Group",
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
	err := runRootCmd(t, "--token", "test-token", cmdGroups, "create", "domain-1", "Group", "--type", "invalid")
	if err == nil {
		t.Fatal("expected unsupported group type error")
	}
	if !strings.Contains(err.Error(), "expected object or principal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

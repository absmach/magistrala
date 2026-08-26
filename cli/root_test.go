// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func clearAtomEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ATOM_URL", "")
	t.Setenv("ATOM_GRAPHQL_URL", "")
	t.Setenv("ATOM_SERVICE_TOKEN", "")
	t.Setenv("ATOM_ADMIN_TOKEN", "")
}

func TestRootCommandContainsAtomBackedCommands(t *testing.T) {
	clearAtomEnv(t)
	cmd := NewRootCmd()
	for _, name := range []string{
		"health", "login", "password", cmdWorkspaces, cmdChannels, cmdGroups,
		"authz", "devices", "gateways", "devicetypes",
	} {
		if subcmd, _, err := cmd.Find([]string{name}); err != nil || subcmd == nil || subcmd.Name() != name {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
}

func TestAuthedCommandsRequireToken(t *testing.T) {
	clearAtomEnv(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{cmdWorkspaces, "list"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected token error")
	}
	if !strings.HasPrefix(err.Error(), "token is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLURLFromEnv(t *testing.T) {
	cases := []struct {
		name       string
		atomURL    string
		graphQLURL string
		want       string
	}{
		{name: "falls back to the local default", want: defaultAtomGraphQLURL},
		{name: "derives the endpoint from ATOM_URL", atomURL: "http://atom:8080/", want: "http://atom:8080/graphql"},
		{name: "trims surrounding whitespace", atomURL: "  http://atom:8080 ", want: "http://atom:8080/graphql"},
		{name: "prefers an explicit endpoint", atomURL: "http://atom:8080", graphQLURL: " http://gw/gql ", want: "http://gw/gql"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATOM_GRAPHQL_URL", tc.graphQLURL)
			if got := graphQLURLFromEnv(tc.atomURL); got != tc.want {
				t.Fatalf("unexpected endpoint: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestGQLInputOmitsEmptyValues(t *testing.T) {
	input := gqlInput{varName: "Demo"}
	input.setString(varAlias, "")
	input.setObject(varAttributes, nil)
	input.setString(varKind, "device")

	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}
	if got, want := string(raw), `{"kind":"device","name":"Demo"}`; got != want {
		t.Fatalf("unexpected input: got %s want %s", got, want)
	}
}

func TestWriteOutputDoesNotEscapeHTML(t *testing.T) {
	cmd := NewRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := writeOutput(cmd, json.RawMessage(`{"name":"A & B <lab>"}`)); err != nil {
		t.Fatalf("failed to write output: %v", err)
	}
	if !strings.Contains(out.String(), `"A & B <lab>"`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestRootCommandConfiguresAtomClient(t *testing.T) {
	clearAtomEnv(t)
	SetAtomClient(nil)
	t.Cleanup(func() { SetAtomClient(nil) })

	cmd := NewRootCmd()
	cmd.SetArgs([]string{cmdWorkspaces, "list"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	// The command itself fails on the missing token, but PersistentPreRun has
	// already run by then, which is what devices/gateways/devicetypes need.
	_ = cmd.Execute()

	if atomClient == nil {
		t.Fatal("expected NewRootCmd to configure the Atom client")
	}
}

func TestAtomBaseURL(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		fallback string
		want     string
	}{
		{name: "strips the GraphQL path", endpoint: "http://atom:8080/graphql", fallback: "http://fb", want: "http://atom:8080"},
		{name: "trims surrounding whitespace", endpoint: " http://atom:8080/graphql ", fallback: "http://fb", want: "http://atom:8080"},
		{name: "keeps the fallback for a foreign path", endpoint: "http://gw/api", fallback: "http://fb", want: "http://fb"},
		{name: "keeps the fallback for a bare path", endpoint: "/graphql", fallback: "http://fb", want: "http://fb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := atomBaseURL(tc.endpoint, tc.fallback); got != tc.want {
				t.Fatalf("unexpected base URL: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestListVariablesUseSharedPaginationFlags(t *testing.T) {
	clearAtomEnv(t)
	cmd := NewRootCmd()
	if err := cmd.PersistentFlags().Set(varLimit, "5"); err != nil {
		t.Fatalf("failed to set limit: %v", err)
	}
	if err := cmd.PersistentFlags().Set(varOffset, "2"); err != nil {
		t.Fatalf("failed to set offset: %v", err)
	}
	t.Cleanup(func() { Limit, Offset = 10, 0 })

	variables := listVariables()
	if got, want := variables[varLimit], uint64(5); got != want {
		t.Errorf("unexpected limit: got %v want %v", got, want)
	}
	if got, want := variables[varOffset], uint64(2); got != want {
		t.Errorf("unexpected offset: got %v want %v", got, want)
	}
}

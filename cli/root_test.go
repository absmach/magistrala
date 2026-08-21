// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"testing"
)

func TestRootCommandContainsAtomBackedCommands(t *testing.T) {
	t.Setenv("MG_ATOM_TOKEN", "")
	t.Setenv("MAGISTRALA_TOKEN", "")
	cmd := NewRootCmd()
	for _, name := range []string{"login", "domains", "devices", "channels", "groups", "authz"} {
		if subcmd, _, err := cmd.Find([]string{name}); err != nil || subcmd == nil || subcmd.Name() != name {
			t.Fatalf("expected command %q to be registered", name)
		}
	}
	if subcmd, _, err := cmd.Find([]string{"clients"}); err == nil && subcmd != nil && subcmd.Name() == "clients" {
		t.Fatal("stale clients command should not be registered after devices rename")
	}
}

func TestAuthedCommandsRequireToken(t *testing.T) {
	t.Setenv("MG_ATOM_TOKEN", "")
	t.Setenv("MAGISTRALA_TOKEN", "")
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"domains", "list"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected token error")
	}
	if got, want := err.Error(), "token is required"; len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("unexpected error: %v", err)
	}
}

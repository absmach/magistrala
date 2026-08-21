// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/stretchr/testify/require"
)

type atomReaderStub struct {
	entity   atom.Entity
	resource atom.Resource
}

func (s atomReaderStub) GetEntity(context.Context, string) (atom.Entity, error) { return s.entity, nil }
func (s atomReaderStub) GetResource(context.Context, string) (atom.Resource, error) {
	return s.resource, nil
}

func TestAtomResolverEnforcesTenantAndAllowListsAttributes(t *testing.T) {
	reader := atomReaderStub{entity: atom.Entity{
		ID: "device-1", Kind: "device", Name: "device", TenantID: "tenant-1",
		Attributes: atom.Attributes{"identity": "device-id", "private_metadata": map[string]any{"secret": "do-not-copy"}},
	}}
	resolver := NewAtomResolver(reader)
	snapshots, err := resolver.Resolve(context.Background(), ResolveRequest{
		Enrollment: Config{DomainID: "tenant-1"},
		Requested:  []BindingRequest{{Slot: "device", Type: "client", ResourceID: "device-1"}},
	})
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, "device-id", snapshots[0].Snapshot["identity"])
	require.NotContains(t, snapshots[0].Snapshot, "private_metadata")
	require.Empty(t, snapshots[0].SecretSnapshot)

	_, err = resolver.Resolve(context.Background(), ResolveRequest{
		Enrollment: Config{DomainID: "tenant-2"},
		Requested:  []BindingRequest{{Slot: "device", Type: "client", ResourceID: "device-1"}},
	})
	require.Error(t, err)
}

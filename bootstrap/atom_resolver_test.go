// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/absmach/magistrala/pkg/errors"
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
		Enrollment: Config{WorkspaceID: "tenant-1"},
		Requested:  []BindingRequest{{Slot: "device", Type: "device", ResourceID: "device-1"}},
	})
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, "device-id", snapshots[0].Snapshot["identity"])
	require.NotContains(t, snapshots[0].Snapshot, "private_metadata")
	require.Empty(t, snapshots[0].SecretSnapshot)

	_, err = resolver.Resolve(context.Background(), ResolveRequest{
		Enrollment: Config{WorkspaceID: "tenant-2"},
		Requested:  []BindingRequest{{Slot: "device", Type: "device", ResourceID: "device-1"}},
	})
	require.Error(t, err)
}

func TestAtomResolverRejectsUnsupportedBindingTypeAsRequestError(t *testing.T) {
	resolver := NewAtomResolver(atomReaderStub{})
	_, err := resolver.Resolve(context.Background(), ResolveRequest{
		Enrollment: Config{WorkspaceID: "tenant-1"},
		Requested:  []BindingRequest{{Slot: "sensor", Type: "client", ResourceID: "device-1"}},
	})
	require.Error(t, err)
	// Must be a typed, nestable error so it survives errors.Wrap in the
	// caller and reaches EncodeError as a *RequestError (400 with a JSON
	// body), instead of an opaque error that falls through to an empty-body
	// 500 - see bootstrap.BindResources and api/http.EncodeError.
	var reqErr *errors.RequestError
	require.ErrorAs(t, err, &reqErr)
}

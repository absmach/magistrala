// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/stretchr/testify/require"
)

type recordingProjector struct{ resources []atom.Resource }

func (*recordingProjector) UpsertTenant(context.Context, atom.Tenant) error { return nil }
func (*recordingProjector) UpsertEntity(context.Context, atom.Entity) error { return nil }
func (*recordingProjector) UpsertGroup(context.Context, atom.Group) error   { return nil }
func (p *recordingProjector) UpsertResource(_ context.Context, resource atom.Resource) error {
	p.resources = append(p.resources, resource)
	return nil
}
func (*recordingProjector) DeleteTenant(context.Context, string) error   { return nil }
func (*recordingProjector) DeleteEntity(context.Context, string) error   { return nil }
func (*recordingProjector) DeleteGroup(context.Context, string) error    { return nil }
func (*recordingProjector) DeleteResource(context.Context, string) error { return nil }

func TestAtomProjectionDoesNotLeakBootstrapSecrets(t *testing.T) {
	ctx := context.Background()
	session := authn.Session{DomainID: "tenant-1"}
	input := bootstrap.Config{ExternalID: "serial-1", ExternalKey: "factory-secret", ClientKey: "private-key", Content: "rendered-secret"}
	saved := input
	saved.ID = "config-1"
	saved.DomainID = session.DomainID
	saved.Name = "device enrollment"

	underlying := mocks.NewService(t)
	underlying.EXPECT().Add(ctx, session, "user-token", input).Return(saved, nil)
	projector := &recordingProjector{}
	svc := bootstrap.WithAtom(underlying, projector)

	_, err := svc.Add(ctx, session, "user-token", input)
	require.NoError(t, err)
	require.Len(t, projector.resources, 1)
	projection := projector.resources[0]
	require.Equal(t, atom.KindBootstrapConfig, projection.Kind)
	require.Equal(t, "tenant-1", projection.TenantID)
	require.NotContains(t, projection.Attributes, "external_id")
	require.NotContains(t, projection.Attributes, "external_key")
	require.NotContains(t, projection.Attributes, "client_key")
	require.NotContains(t, projection.Attributes, "content")
	require.NotContains(t, projection.Attributes, "render_context")
}

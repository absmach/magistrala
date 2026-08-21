// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/pkg/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/stretchr/testify/require"
)

type recordingProjector struct {
	resources  []atom.Resource
	upsertErr  error
	deleteErr  error
	deletedIDs []string
}

func (*recordingProjector) UpsertTenant(context.Context, atom.Tenant) error { return nil }
func (*recordingProjector) UpsertEntity(context.Context, atom.Entity) error { return nil }
func (*recordingProjector) UpsertGroup(context.Context, atom.Group) error   { return nil }
func (p *recordingProjector) UpsertResource(_ context.Context, resource atom.Resource) error {
	p.resources = append(p.resources, resource)
	return p.upsertErr
}
func (*recordingProjector) DeleteTenant(context.Context, string) error { return nil }
func (*recordingProjector) DeleteEntity(context.Context, string) error { return nil }
func (*recordingProjector) DeleteGroup(context.Context, string) error  { return nil }
func (p *recordingProjector) DeleteResource(_ context.Context, id string) error {
	p.deletedIDs = append(p.deletedIDs, id)
	return p.deleteErr
}

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

func TestAtomProjectionFailuresAreReturned(t *testing.T) {
	ctx := context.Background()
	session := authn.Session{DomainID: "tenant-1"}
	projectionErr := stderrors.New("atom unavailable")
	config := bootstrap.Config{ID: "config-1", DomainID: session.DomainID, Name: "enrollment"}
	profile := bootstrap.Profile{ID: "profile-1", DomainID: session.DomainID, Name: "profile"}

	t.Run("add", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().Add(ctx, session, "user-token", config).Return(config, nil)
		_, err := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr}).Add(ctx, session, "user-token", config)
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("update", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().Update(ctx, session, config).Return(nil)
		underlying.EXPECT().View(ctx, session, config.ID).Return(config, nil)
		err := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr}).Update(ctx, session, config)
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("update certificate", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().UpdateCert(ctx, session, config.ID, "cert", "key", "ca").Return(config, nil)
		_, err := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr}).UpdateCert(ctx, session, config.ID, "cert", "key", "ca")
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("enable and disable", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().EnableConfig(ctx, session, config.ID).Return(config, nil)
		underlying.EXPECT().DisableConfig(ctx, session, config.ID).Return(config, nil)
		svc := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr})
		_, err := svc.EnableConfig(ctx, session, config.ID)
		require.ErrorIs(t, err, projectionErr)
		_, err = svc.DisableConfig(ctx, session, config.ID)
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("create and update profile", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().CreateProfile(ctx, session, profile).Return(profile, nil)
		underlying.EXPECT().UpdateProfile(ctx, session, profile).Return(profile, nil)
		svc := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr})
		_, err := svc.CreateProfile(ctx, session, profile)
		require.ErrorIs(t, err, projectionErr)
		_, err = svc.UpdateProfile(ctx, session, profile)
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("assign profile", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().AssignProfile(ctx, session, config.ID, profile.ID).Return(nil)
		underlying.EXPECT().View(ctx, session, config.ID).Return(config, nil)
		err := bootstrap.WithAtom(underlying, &recordingProjector{upsertErr: projectionErr}).AssignProfile(ctx, session, config.ID, profile.ID)
		require.ErrorIs(t, err, projectionErr)
	})

	t.Run("delete config and profile", func(t *testing.T) {
		underlying := mocks.NewService(t)
		underlying.EXPECT().Remove(ctx, session, config.ID).Return(nil)
		underlying.EXPECT().DeleteProfile(ctx, session, profile.ID).Return(nil)
		svc := bootstrap.WithAtom(underlying, &recordingProjector{deleteErr: projectionErr})
		err := svc.Remove(ctx, session, config.ID)
		require.ErrorIs(t, err, projectionErr)
		err = svc.DeleteProfile(ctx, session, profile.ID)
		require.ErrorIs(t, err, projectionErr)
	})
}

func TestReconcileAtomBackfillsConfigsAndProfiles(t *testing.T) {
	ctx := context.Background()
	config := bootstrap.Config{ID: "config-1", DomainID: "tenant-1", Name: "enrollment"}
	profile := bootstrap.Profile{ID: "profile-1", DomainID: "tenant-1", Name: "profile"}
	underlying := mocks.NewService(t)
	underlying.EXPECT().List(ctx, authn.Session{}, bootstrap.Filter{}, uint64(0), uint64(100)).Return(bootstrap.ConfigsPage{
		Total: 1, Configs: []bootstrap.Config{config},
	}, nil)
	underlying.EXPECT().ListProfiles(ctx, authn.Session{}, uint64(0), uint64(100), "").Return(bootstrap.ProfilesPage{
		Total: 1, Profiles: []bootstrap.Profile{profile},
	}, nil)
	projector := &recordingProjector{}

	require.NoError(t, bootstrap.ReconcileAtom(ctx, underlying, projector))
	require.ElementsMatch(t, []string{config.ID, profile.ID}, []string{projector.resources[0].ID, projector.resources[1].ID})
}

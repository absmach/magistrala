// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/policies"
)

// AtomAuthorizationMiddleware authorizes bootstrap management operations with
// Atom. The unauthenticated device bootstrap endpoint remains protected by its
// external ID/key pair in the domain service.
func AtomAuthorizationMiddleware(svc bootstrap.Service, authorizer atom.Authorizer) bootstrap.Service {
	return &atomAuthorizationMiddleware{Service: svc, authorizer: authorizer}
}

type atomAuthorizationMiddleware struct {
	bootstrap.Service
	authorizer atom.Authorizer
}

func (am *atomAuthorizationMiddleware) tenant(ctx context.Context, session authn.Session, action string) error {
	return atom.Authorize(ctx, am.authorizer, session, action, policies.WorkspaceType, session.WorkspaceID, policies.WorkspaceType)
}

func (am *atomAuthorizationMiddleware) resource(ctx context.Context, session authn.Session, action, id, kind string) error {
	return atom.Authorize(ctx, am.authorizer, session, action, kind, id, kind)
}

func (am *atomAuthorizationMiddleware) Add(ctx context.Context, session authn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	if err := am.tenant(ctx, session, "write"); err != nil {
		return bootstrap.Config{}, err
	}
	return am.Service.Add(ctx, session, token, cfg)
}

func (am *atomAuthorizationMiddleware) View(ctx context.Context, session authn.Session, id string) (bootstrap.Config, error) {
	if err := am.resource(ctx, session, "read", id, atom.KindBootstrapConfig); err != nil {
		return bootstrap.Config{}, err
	}
	return am.Service.View(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) Update(ctx context.Context, session authn.Session, cfg bootstrap.Config) error {
	if err := am.resource(ctx, session, "write", cfg.ID, atom.KindBootstrapConfig); err != nil {
		return err
	}
	return am.Service.Update(ctx, session, cfg)
}

func (am *atomAuthorizationMiddleware) UpdateCert(ctx context.Context, session authn.Session, id, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	if err := am.resource(ctx, session, "write", id, atom.KindBootstrapConfig); err != nil {
		return bootstrap.Config{}, err
	}
	return am.Service.UpdateCert(ctx, session, id, clientCert, clientKey, caCert)
}

func (am *atomAuthorizationMiddleware) List(ctx context.Context, session authn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	// Listing reads, so it checks "read" rather than "write" or a distinct
	// "list" capability. This matches alarms and re: their list operations
	// resolve through permission.yaml to a "*_read_permission" string, which
	// CapabilityName collapses to "read" — no tenant role is ever granted a
	// standalone "list" capability.
	if err := am.tenant(ctx, session, "read"); err != nil {
		return bootstrap.ConfigsPage{}, err
	}
	return am.Service.List(ctx, session, filter, offset, limit)
}

func (am *atomAuthorizationMiddleware) Remove(ctx context.Context, session authn.Session, id string) error {
	if err := am.resource(ctx, session, "delete", id, atom.KindBootstrapConfig); err != nil {
		return err
	}
	return am.Service.Remove(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) EnableConfig(ctx context.Context, session authn.Session, id string) (bootstrap.Config, error) {
	if err := am.resource(ctx, session, "write", id, atom.KindBootstrapConfig); err != nil {
		return bootstrap.Config{}, err
	}
	return am.Service.EnableConfig(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) DisableConfig(ctx context.Context, session authn.Session, id string) (bootstrap.Config, error) {
	if err := am.resource(ctx, session, "write", id, atom.KindBootstrapConfig); err != nil {
		return bootstrap.Config{}, err
	}
	return am.Service.DisableConfig(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) CreateProfile(ctx context.Context, session authn.Session, p bootstrap.Profile) (bootstrap.Profile, error) {
	if err := am.tenant(ctx, session, "write"); err != nil {
		return bootstrap.Profile{}, err
	}
	return am.Service.CreateProfile(ctx, session, p)
}

func (am *atomAuthorizationMiddleware) ViewProfile(ctx context.Context, session authn.Session, id string) (bootstrap.Profile, error) {
	if err := am.resource(ctx, session, "read", id, atom.KindBootstrapProfile); err != nil {
		return bootstrap.Profile{}, err
	}
	return am.Service.ViewProfile(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) UpdateProfile(ctx context.Context, session authn.Session, p bootstrap.Profile) (bootstrap.Profile, error) {
	if err := am.resource(ctx, session, "write", p.ID, atom.KindBootstrapProfile); err != nil {
		return bootstrap.Profile{}, err
	}
	return am.Service.UpdateProfile(ctx, session, p)
}

func (am *atomAuthorizationMiddleware) ListProfiles(ctx context.Context, session authn.Session, offset, limit uint64, name string) (bootstrap.ProfilesPage, error) {
	// See List: this reads, so it checks "read".
	if err := am.tenant(ctx, session, "read"); err != nil {
		return bootstrap.ProfilesPage{}, err
	}
	return am.Service.ListProfiles(ctx, session, offset, limit, name)
}

func (am *atomAuthorizationMiddleware) DeleteProfile(ctx context.Context, session authn.Session, id string) error {
	if err := am.resource(ctx, session, "delete", id, atom.KindBootstrapProfile); err != nil {
		return err
	}
	return am.Service.DeleteProfile(ctx, session, id)
}

func (am *atomAuthorizationMiddleware) AssignProfile(ctx context.Context, session authn.Session, configID, profileID string) error {
	if err := am.resource(ctx, session, "write", configID, atom.KindBootstrapConfig); err != nil {
		return err
	}
	if err := am.resource(ctx, session, "read", profileID, atom.KindBootstrapProfile); err != nil {
		return err
	}
	return am.Service.AssignProfile(ctx, session, configID, profileID)
}

func (am *atomAuthorizationMiddleware) BindResources(ctx context.Context, session authn.Session, token, configID string, bindings []bootstrap.BindingRequest) error {
	// Rewriting an enrollment's bindings is a write to that enrollment, so it
	// is checked per config exactly as RefreshBindings is. A tenant-wide check
	// alone would let a principal denied write on this specific config still
	// replace its bindings.
	if err := am.resource(ctx, session, "write", configID, atom.KindBootstrapConfig); err != nil {
		return err
	}
	return am.Service.BindResources(ctx, session, token, configID, bindings)
}

func (am *atomAuthorizationMiddleware) ListBindings(ctx context.Context, session authn.Session, configID string) ([]bootstrap.BindingSnapshot, error) {
	if err := am.resource(ctx, session, "read", configID, atom.KindBootstrapConfig); err != nil {
		return nil, err
	}
	return am.Service.ListBindings(ctx, session, configID)
}

func (am *atomAuthorizationMiddleware) RefreshBindings(ctx context.Context, session authn.Session, token, configID string) error {
	if err := am.resource(ctx, session, "write", configID, atom.KindBootstrapConfig); err != nil {
		return err
	}
	return am.Service.RefreshBindings(ctx, session, token, configID)
}

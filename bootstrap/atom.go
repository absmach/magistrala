// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
)

type atomService struct {
	Service
	projector atom.Projector
}

// WithAtom projects non-secret bootstrap metadata into Atom. PostgreSQL remains
// authoritative for enrollments, templates, certificates, keys and snapshots.
func WithAtom(svc Service, projector atom.Projector) Service {
	if projector == nil {
		return svc
	}
	return &atomService{Service: svc, projector: projector}
}

func (svc *atomService) Add(ctx context.Context, session authn.Session, token string, cfg Config) (Config, error) {
	saved, err := svc.Service.Add(ctx, session, token, cfg)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(saved))
	}
	return saved, err
}

func (svc *atomService) Update(ctx context.Context, session authn.Session, cfg Config) error {
	if err := svc.Service.Update(ctx, session, cfg); err != nil {
		return err
	}
	updated, err := svc.Service.View(ctx, session, cfg.ID)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(updated))
	}
	return nil
}

func (svc *atomService) UpdateCert(ctx context.Context, session authn.Session, id, clientCert, clientKey, caCert string) (Config, error) {
	updated, err := svc.Service.UpdateCert(ctx, session, id, clientCert, clientKey, caCert)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(updated))
	}
	return updated, err
}

func (svc *atomService) Remove(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.Remove(ctx, session, id); err != nil {
		return err
	}
	_ = svc.projector.DeleteResource(ctx, id)
	return nil
}

func (svc *atomService) EnableConfig(ctx context.Context, session authn.Session, id string) (Config, error) {
	updated, err := svc.Service.EnableConfig(ctx, session, id)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(updated))
	}
	return updated, err
}

func (svc *atomService) DisableConfig(ctx context.Context, session authn.Session, id string) (Config, error) {
	updated, err := svc.Service.DisableConfig(ctx, session, id)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(updated))
	}
	return updated, err
}

func (svc *atomService) CreateProfile(ctx context.Context, session authn.Session, p Profile) (Profile, error) {
	saved, err := svc.Service.CreateProfile(ctx, session, p)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, profileProjection(saved))
	}
	return saved, err
}

func (svc *atomService) UpdateProfile(ctx context.Context, session authn.Session, p Profile) (Profile, error) {
	updated, err := svc.Service.UpdateProfile(ctx, session, p)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, profileProjection(updated))
	}
	return updated, err
}

func (svc *atomService) DeleteProfile(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.DeleteProfile(ctx, session, id); err != nil {
		return err
	}
	_ = svc.projector.DeleteResource(ctx, id)
	return nil
}

func (svc *atomService) AssignProfile(ctx context.Context, session authn.Session, configID, profileID string) error {
	if err := svc.Service.AssignProfile(ctx, session, configID, profileID); err != nil {
		return err
	}
	updated, err := svc.Service.View(ctx, session, configID)
	if err == nil {
		_ = svc.projector.UpsertResource(ctx, configProjection(updated))
	}
	return nil
}

func configProjection(cfg Config) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID: cfg.ID, Kind: atom.KindBootstrapConfig, Name: cfg.Name,
		TenantID: cfg.DomainID, Status: cfg.Status.String(),
	})
	// Only non-secret relationship metadata is projected. External identifiers,
	// keys, certificates, rendered content and render context stay in PostgreSQL.
	if cfg.ProfileID != "" {
		res.Attributes["profile_id"] = cfg.ProfileID
	}
	return res
}

func profileProjection(p Profile) atom.Resource {
	return atom.ResourceFromFields(atom.ObjectFields{
		ID: p.ID, Kind: atom.KindBootstrapProfile, Name: p.Name,
		TenantID: p.DomainID, Description: p.Description,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		Metadata: map[string]any{
			"content_format": p.ContentFormat,
			"content_type":   p.ContentType,
			"version":        p.Version,
		},
	})
}

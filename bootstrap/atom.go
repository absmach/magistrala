// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/pkg/atom"
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
	if err != nil {
		return saved, err
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(saved)); err != nil {
		return saved, atomProjectionError("upsert config", saved.ID, err)
	}
	return saved, nil
}

func (svc *atomService) Update(ctx context.Context, session authn.Session, cfg Config) error {
	if err := svc.Service.Update(ctx, session, cfg); err != nil {
		return err
	}
	updated, err := svc.Service.View(ctx, session, cfg.ID)
	if err != nil {
		return fmt.Errorf("reload bootstrap config %q for Atom projection: %w", cfg.ID, err)
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(updated)); err != nil {
		return atomProjectionError("upsert config", updated.ID, err)
	}
	return nil
}

func (svc *atomService) UpdateCert(ctx context.Context, session authn.Session, id, clientCert, clientKey, caCert string) (Config, error) {
	updated, err := svc.Service.UpdateCert(ctx, session, id, clientCert, clientKey, caCert)
	if err != nil {
		return updated, err
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(updated)); err != nil {
		return updated, atomProjectionError("upsert config", updated.ID, err)
	}
	return updated, nil
}

func (svc *atomService) Remove(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.Remove(ctx, session, id); err != nil {
		return err
	}
	if err := svc.projector.DeleteResource(ctx, id); err != nil {
		return atomProjectionError("delete config", id, err)
	}
	return nil
}

func (svc *atomService) EnableConfig(ctx context.Context, session authn.Session, id string) (Config, error) {
	updated, err := svc.Service.EnableConfig(ctx, session, id)
	if err != nil {
		return updated, err
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(updated)); err != nil {
		return updated, atomProjectionError("upsert config", updated.ID, err)
	}
	return updated, nil
}

func (svc *atomService) DisableConfig(ctx context.Context, session authn.Session, id string) (Config, error) {
	updated, err := svc.Service.DisableConfig(ctx, session, id)
	if err != nil {
		return updated, err
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(updated)); err != nil {
		return updated, atomProjectionError("upsert config", updated.ID, err)
	}
	return updated, nil
}

func (svc *atomService) CreateProfile(ctx context.Context, session authn.Session, p Profile) (Profile, error) {
	saved, err := svc.Service.CreateProfile(ctx, session, p)
	if err != nil {
		return saved, err
	}
	if err := svc.projector.UpsertResource(ctx, profileProjection(saved)); err != nil {
		return saved, atomProjectionError("upsert profile", saved.ID, err)
	}
	return saved, nil
}

func (svc *atomService) UpdateProfile(ctx context.Context, session authn.Session, p Profile) (Profile, error) {
	updated, err := svc.Service.UpdateProfile(ctx, session, p)
	if err != nil {
		return updated, err
	}
	if err := svc.projector.UpsertResource(ctx, profileProjection(updated)); err != nil {
		return updated, atomProjectionError("upsert profile", updated.ID, err)
	}
	return updated, nil
}

func (svc *atomService) DeleteProfile(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.DeleteProfile(ctx, session, id); err != nil {
		return err
	}
	if err := svc.projector.DeleteResource(ctx, id); err != nil {
		return atomProjectionError("delete profile", id, err)
	}
	return nil
}

func (svc *atomService) AssignProfile(ctx context.Context, session authn.Session, configID, profileID string) error {
	if err := svc.Service.AssignProfile(ctx, session, configID, profileID); err != nil {
		return err
	}
	updated, err := svc.Service.View(ctx, session, configID)
	if err != nil {
		return fmt.Errorf("reload bootstrap config %q for Atom projection: %w", configID, err)
	}
	if err := svc.projector.UpsertResource(ctx, configProjection(updated)); err != nil {
		return atomProjectionError("upsert config", updated.ID, err)
	}
	return nil
}

const atomProjectionPageSize uint64 = 100

type atomResourceLister interface {
	ListResources(context.Context, atom.Query) (atom.ResourceList, error)
}

// ReconcileAtom backfills Bootstrap's non-secret Atom projections and removes
// projections whose PostgreSQL records no longer exist. It must run before the
// authorization decorator so its empty session is never externally reachable.
func ReconcileAtom(ctx context.Context, svc Service, projector atom.Projector) error {
	if svc == nil {
		return fmt.Errorf("reconcile bootstrap Atom projections: service is nil")
	}
	if projector == nil {
		return fmt.Errorf("reconcile bootstrap Atom projections: projector is nil")
	}

	configIDs, err := reconcileConfigs(ctx, svc, projector)
	if err != nil {
		return err
	}
	profileIDs, err := reconcileProfiles(ctx, svc, projector)
	if err != nil {
		return err
	}
	if lister, ok := projector.(atomResourceLister); ok {
		if err := deleteStaleAtomResources(ctx, projector, lister, atom.KindBootstrapConfig, configIDs); err != nil {
			return err
		}
		if err := deleteStaleAtomResources(ctx, projector, lister, atom.KindBootstrapProfile, profileIDs); err != nil {
			return err
		}
	}
	return nil
}

func reconcileConfigs(ctx context.Context, svc Service, projector atom.Projector) (map[string]struct{}, error) {
	ids := map[string]struct{}{}
	for offset := uint64(0); ; {
		page, err := svc.List(ctx, authn.Session{}, Filter{}, offset, atomProjectionPageSize)
		if err != nil {
			return nil, fmt.Errorf("list bootstrap configs for Atom reconciliation: %w", err)
		}
		for _, cfg := range page.Configs {
			if err := projector.UpsertResource(ctx, configProjection(cfg)); err != nil {
				return nil, atomProjectionError("backfill config", cfg.ID, err)
			}
			ids[cfg.ID] = struct{}{}
		}
		count := uint64(len(page.Configs))
		if count == 0 || offset+count >= page.Total {
			return ids, nil
		}
		offset += count
	}
}

func reconcileProfiles(ctx context.Context, svc Service, projector atom.Projector) (map[string]struct{}, error) {
	ids := map[string]struct{}{}
	for offset := uint64(0); ; {
		page, err := svc.ListProfiles(ctx, authn.Session{}, offset, atomProjectionPageSize, "")
		if err != nil {
			return nil, fmt.Errorf("list bootstrap profiles for Atom reconciliation: %w", err)
		}
		for _, profile := range page.Profiles {
			if err := projector.UpsertResource(ctx, profileProjection(profile)); err != nil {
				return nil, atomProjectionError("backfill profile", profile.ID, err)
			}
			ids[profile.ID] = struct{}{}
		}
		count := uint64(len(page.Profiles))
		if count == 0 || offset+count >= page.Total {
			return ids, nil
		}
		offset += count
	}
}

func deleteStaleAtomResources(ctx context.Context, projector atom.Projector, lister atomResourceLister, kind string, localIDs map[string]struct{}) error {
	var staleIDs []string
	for offset := uint64(0); ; {
		page, err := lister.ListResources(ctx, atom.Query{Kind: kind, Limit: atomProjectionPageSize, Offset: offset})
		if err != nil {
			return fmt.Errorf("list Atom %s resources for reconciliation: %w", kind, err)
		}
		for _, resource := range page.Items {
			if _, ok := localIDs[resource.ID]; !ok {
				staleIDs = append(staleIDs, resource.ID)
			}
		}
		count := uint64(len(page.Items))
		if count == 0 || offset+count >= page.Total {
			break
		}
		offset += count
	}
	for _, id := range staleIDs {
		if err := projector.DeleteResource(ctx, id); err != nil {
			return atomProjectionError("delete stale resource", id, err)
		}
	}
	return nil
}

func atomProjectionError(operation, id string, err error) error {
	return fmt.Errorf("Atom projection %s for bootstrap resource %q: %w", operation, id, err)
}

func configProjection(cfg Config) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID: cfg.ID, Kind: atom.KindBootstrapConfig, Name: cfg.Name,
		TenantID: cfg.WorkspaceID, Status: cfg.Status.String(),
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
		TenantID: p.WorkspaceID, Description: p.Description,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		Metadata: map[string]any{
			"content_format": p.ContentFormat,
			"content_type":   p.ContentType,
			"version":        p.Version,
		},
	})
}

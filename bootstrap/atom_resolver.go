// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

type atomResourceReader interface {
	GetEntity(context.Context, string) (atom.Entity, error)
	GetResource(context.Context, string) (atom.Resource, error)
}

type atomResolver struct{ client atomResourceReader }

// NewAtomResolver resolves device and channel bindings from Atom. It copies
// only explicitly allow-listed, non-secret fields into bootstrap snapshots.
func NewAtomResolver(client atomResourceReader) BindingResolver {
	return &atomResolver{client: client}
}

func (r *atomResolver) Resolve(ctx context.Context, req ResolveRequest) ([]BindingSnapshot, error) {
	result := make([]BindingSnapshot, 0, len(req.Requested))
	for _, binding := range req.Requested {
		var snapshot BindingSnapshot
		var err error
		switch binding.Type {
		case "device":
			snapshot, err = r.entity(ctx, req.Enrollment.WorkspaceID, binding)
		case "channel":
			snapshot, err = r.resource(ctx, req.Enrollment.WorkspaceID, binding)
		default:
			err = errors.NewRequestError(fmt.Sprintf("unsupported binding type %q", binding.Type))
		}
		if err != nil {
			return nil, err
		}
		result = append(result, snapshot)
	}
	return result, nil
}

func (r *atomResolver) entity(ctx context.Context, tenantID string, binding BindingRequest) (BindingSnapshot, error) {
	entity, err := r.client.GetEntity(ctx, binding.ResourceID)
	if err != nil || entity.TenantID != tenantID || entity.Kind != "device" {
		return BindingSnapshot{}, errors.Wrap(svcerr.ErrNotFound, fmt.Errorf("device %q not found in tenant", binding.ResourceID))
	}
	values := map[string]any{snapshotKeyID: entity.ID, snapshotKeyName: entity.Name, snapshotKeyWorkspaceID: entity.TenantID}
	copyStringAttribute(values, entity.Attributes, snapshotKeyIdentity)
	copyStringAttribute(values, entity.Attributes, snapshotKeyRoute)
	return BindingSnapshot{Slot: binding.Slot, Type: binding.Type, ResourceID: binding.ResourceID, Snapshot: values, UpdatedAt: time.Now().UTC()}, nil
}

func (r *atomResolver) resource(ctx context.Context, tenantID string, binding BindingRequest) (BindingSnapshot, error) {
	resource, err := r.client.GetResource(ctx, binding.ResourceID)
	if err != nil || resource.TenantID != tenantID || resource.Kind != atom.KindChannel {
		return BindingSnapshot{}, errors.Wrap(svcerr.ErrNotFound, fmt.Errorf("channel %q not found in tenant", binding.ResourceID))
	}
	values := map[string]any{snapshotKeyID: resource.ID, snapshotKeyName: resource.Name, snapshotKeyWorkspaceID: resource.TenantID}
	copyStringAttribute(values, resource.Attributes, snapshotKeyRoute)
	copyStringAttribute(values, resource.Attributes, snapshotKeyTopic)
	return BindingSnapshot{Slot: binding.Slot, Type: binding.Type, ResourceID: binding.ResourceID, Snapshot: values, UpdatedAt: time.Now().UTC()}, nil
}

func copyStringAttribute(dst map[string]any, src atom.Attributes, key string) {
	if value, ok := src[key].(string); ok && value != "" {
		dst[key] = value
	}
}

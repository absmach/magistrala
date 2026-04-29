// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
)

var _ BindingResolver = (*sdkResolver)(nil)

type sdkResolver struct {
	sdk mgsdk.SDK
}

// NewSDKResolver returns a BindingResolver that validates resources against
// the Magistrala clients and channels services using the SDK. This resolver
// is called only at binding time; the render path must never call it.
func NewSDKResolver(sdk mgsdk.SDK) BindingResolver {
	return &sdkResolver{sdk: sdk}
}

func (r *sdkResolver) Resolve(ctx context.Context, req ResolveRequest) ([]BindingSnapshot, error) {
	var snapshots []BindingSnapshot

	for _, br := range req.Requested {
		snap, err := r.resolveOne(ctx, req.Enrollment.DomainID, req.Token, br)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snap)
	}

	return snapshots, nil
}

func (r *sdkResolver) resolveOne(ctx context.Context, domainID, token string, br BindingRequest) (BindingSnapshot, error) {
	switch br.Type {
	case "client":
		return r.resolveClient(ctx, domainID, token, br)
	case "channel":
		return r.resolveChannel(ctx, domainID, token, br)
	default:
		return BindingSnapshot{}, fmt.Errorf("unsupported binding type %q", br.Type)
	}
}

func (r *sdkResolver) resolveClient(ctx context.Context, domainID, token string, br BindingRequest) (BindingSnapshot, error) {
	client, sdkErr := r.sdk.Client(ctx, br.ResourceID, domainID, token)
	if sdkErr != nil {
		return BindingSnapshot{}, errors.Wrap(svcerr.ErrNotFound,
			fmt.Errorf("client %q not found: %s", br.ResourceID, sdkErr))
	}

	snapshot := map[string]any{
		"id":   client.ID,
		"name": client.Name,
	}
	if client.Credentials.Identity != "" {
		snapshot["identity"] = client.Credentials.Identity
	}
	if client.DomainID != "" {
		snapshot["domain_id"] = client.DomainID
	}

	secret := map[string]any{}
	if client.Credentials.Secret != "" {
		secret["secret"] = client.Credentials.Secret
	}

	return BindingSnapshot{
		Slot:           br.Slot,
		Type:           br.Type,
		ResourceID:     br.ResourceID,
		Snapshot:       snapshot,
		SecretSnapshot: secret,
		UpdatedAt:      time.Now().UTC(),
	}, nil
}

func (r *sdkResolver) resolveChannel(ctx context.Context, domainID, token string, br BindingRequest) (BindingSnapshot, error) {
	channel, sdkErr := r.sdk.Channel(ctx, br.ResourceID, domainID, token)
	if sdkErr != nil {
		return BindingSnapshot{}, errors.Wrap(svcerr.ErrNotFound,
			fmt.Errorf("channel %q not found: %s", br.ResourceID, sdkErr))
	}

	snapshot := map[string]any{
		"id":   channel.ID,
		"name": channel.Name,
	}
	if channel.Route != "" {
		snapshot["topic"] = channel.Route
	}
	if channel.DomainID != "" {
		snapshot["domain_id"] = channel.DomainID
	}
	if channel.Metadata != nil {
		snapshot["metadata"] = channel.Metadata
	}

	return BindingSnapshot{
		Slot:       br.Slot,
		Type:       br.Type,
		ResourceID: br.ResourceID,
		Snapshot:   snapshot,
		UpdatedAt:  time.Now().UTC(),
	}, nil
}

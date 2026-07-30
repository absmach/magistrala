// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import "context"

type Projector interface {
	UpsertTenant(ctx context.Context, tenant Tenant) error
	UpsertEntity(ctx context.Context, entity Entity) error
	UpsertGroup(ctx context.Context, group Group) error
	UpsertResource(ctx context.Context, resource Resource) error
	DeleteTenant(ctx context.Context, id string) error
	DeleteEntity(ctx context.Context, id string) error
	DeleteGroup(ctx context.Context, id string) error
	DeleteResource(ctx context.Context, id string) error
}

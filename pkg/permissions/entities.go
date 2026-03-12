// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package permissions

import (
	"fmt"

	"github.com/absmach/supermq/pkg/errors"
)

type (
	EntitiesPermission                       map[string]map[string]Permission
	EntitiesOperationDetails[K OperationKey] map[string]map[K]OperationDetails
)

type EntitiesOperations[K OperationKey] interface {
	GetPermission(et string, op K) (Permission, error)
	GetPermissionAndRequired(et string, op K) (Permission, bool, error)
	OperationName(et string, op K) string
	PATOperationName(et string, op K) string
	Validate() error
	AddEntityOperations(et string, ops Operations[K]) error
	RemoveEntityOperations(et string, ops Operations[K]) error
}

var ErrEntityTypeNotFound = errors.New("entity type not found")

type entitiesOperations[K OperationKey] map[string]Operations[K]

func NewEntitiesOperations[K OperationKey](entitiesPermission EntitiesPermission, entitiesOperationDetails EntitiesOperationDetails[K], filterEntities ...string) (EntitiesOperations[K], error) {
	if len(filterEntities) == 0 {
		return newEntitiesOperations(entitiesPermission, entitiesOperationDetails)
	}

	filterSet := make(map[string]struct{}, len(filterEntities))
	for _, entity := range filterEntities {
		filterSet[entity] = struct{}{}
	}

	filteredDetails := make(EntitiesOperationDetails[K])
	filteredPerms := make(EntitiesPermission)
	for entity := range filterSet {
		if opDetails, ok := entitiesOperationDetails[entity]; ok {
			filteredDetails[entity] = opDetails
		}
		if perms, ok := entitiesPermission[entity]; ok {
			filteredPerms[entity] = perms
		}
	}

	return newEntitiesOperations(filteredPerms, filteredDetails)
}

func newEntitiesOperations[K OperationKey](entitiesPermission EntitiesPermission, entitiesOperationDetails EntitiesOperationDetails[K]) (EntitiesOperations[K], error) {
	eops := make(entitiesOperations[K])
	for entity, opDetails := range entitiesOperationDetails {
		opPerm, ok := entitiesPermission[entity]
		if !ok {
			return nil, fmt.Errorf("%s entity permission not found ", entity)
		}
		ops, err := NewOperations(opDetails, opPerm)
		if err != nil {
			return nil, fmt.Errorf("failed to create new operations for %s entity: %w", entity, err)
		}
		eops[entity] = ops
	}
	return eops, nil
}

// Implement the interface.
func (eo entitiesOperations[K]) GetPermission(et string, op K) (Permission, error) {
	if ops, ok := eo[et]; ok {
		return ops.GetPermission(op)
	}
	return Permission(""), ErrEntityTypeNotFound
}

func (eo entitiesOperations[K]) GetPermissionAndRequired(et string, op K) (Permission, bool, error) {
	if ops, ok := eo[et]; ok {
		return ops.GetPermissionAndRequired(op)
	}
	return Permission(""), false, ErrEntityTypeNotFound
}

func (eo entitiesOperations[K]) OperationName(et string, op K) string {
	if ops, ok := eo[et]; ok {
		return ops.OperationName(op)
	}
	return ""
}

func (eo entitiesOperations[K]) PATOperationName(et string, op K) string {
	if ops, ok := eo[et]; ok {
		return ops.PATOperationName(op)
	}
	return ""
}

func (eo entitiesOperations[K]) Validate() error {
	for et, ops := range eo {
		if err := ops.Validate(); err != nil {
			return fmt.Errorf("entity type %s failed to validate: %w", et, err)
		}
	}
	return nil
}

func (eo entitiesOperations[K]) AddEntityOperations(et string, ops Operations[K]) error {
	if entityOperation, ok := eo[et]; ok {
		return entityOperation.Merge(ops)
	}
	eo[et] = ops
	return nil
}

func (eo entitiesOperations[K]) RemoveEntityOperations(et string, ops Operations[K]) error {
	if entityOperation, ok := eo[et]; ok {
		return entityOperation.Remove(ops)
	}
	return nil
}

// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package permissions

import (
	"fmt"

	"github.com/absmach/supermq/pkg/errors"
)

var (
	ErrMergeInvalidOperations  = errors.New("failed to merge: invalid operations type")
	ErrRemoveInvalidOperations = errors.New("failed to remove: invalid operations type")
)

type Permission string

func (p Permission) String() string {
	return string(p)
}

type Operation int

type ExternalOperation int

type RoleOperation int

type OperationKey interface {
	Operation | ExternalOperation | RoleOperation
}

type OperationName[K OperationKey] map[K]string

type OperationDetails struct {
	Name               string
	PATOpName          string
	PermissionRequired bool
}

type operations[K OperationKey] struct {
	opPermission map[K]Permission
	opDetails    map[K]OperationDetails
}

type Operations[K OperationKey] interface {
	GetPermission(op K) (Permission, error)
	GetPermissionAndRequired(op K) (Permission, bool, error)
	OperationName(op K) string
	PATOperationName(op K) string
	Validate() error
	Merge(nops Operations[K]) error
	Remove(rops Operations[K]) error
}

func NewOperations[K OperationKey](opdetails map[K]OperationDetails, opnamePerm map[string]Permission) (Operations[K], error) {
	ops := newEmptyOperations(opdetails)

	if err := ops.addOperationPermission(opnamePerm); err != nil {
		return nil, err
	}
	if err := ops.Validate(); err != nil {
		return nil, err
	}
	return &ops, nil
}

func newEmptyOperations[K OperationKey](opdetails map[K]OperationDetails) operations[K] {
	return operations[K]{
		opPermission: make(map[K]Permission),
		opDetails:    opdetails,
	}
}

func (ops *operations[K]) OperationName(op K) string {
	opDetail, ok := ops.opDetails[op]
	if !ok {
		return fmt.Sprintf("UnknownOperation(%v)", op)
	}
	return opDetail.Name
}

func (ops *operations[K]) PATOperationName(op K) string {
	opDetail, ok := ops.opDetails[op]
	if !ok {
		return fmt.Sprintf("UnknownOperation(%v)", op)
	}
	if opDetail.PATOpName != "" {
		return opDetail.PATOpName
	}
	return opDetail.Name
}

func (ops *operations[K]) addOperationPermission(opnamePerm map[string]Permission) error {
	for op, opd := range ops.opDetails {
		if opd.PermissionRequired {
			perm, ok := opnamePerm[opd.Name]
			if !ok {
				return fmt.Errorf("permission related to operation name %s not found", opd.Name)
			}
			ops.opPermission[op] = perm
		}
	}
	return nil
}

func (ops *operations[K]) Validate() error {
	for op, opd := range ops.opDetails {
		if opd.PermissionRequired {
			if _, ok := ops.opPermission[op]; !ok {
				return fmt.Errorf("permission related to operation name %s not found", opd.Name)
			}
		}
	}
	return nil
}

func (ops *operations[K]) GetPermission(op K) (Permission, error) {
	if perm, ok := ops.opPermission[op]; ok {
		return perm, nil
	}
	return "", fmt.Errorf("operation %s doesn't have any permissions", ops.OperationName(op))
}

func (ops *operations[K]) GetPermissionAndRequired(op K) (Permission, bool, error) {
	opd, ok := ops.opDetails[op]
	if !ok {
		return "", false, fmt.Errorf("operation not found %s", ops.OperationName(op))
	}
	perm, ok := ops.opPermission[op]
	if opd.PermissionRequired && !ok {
		return "", false, fmt.Errorf("operation %s doesn't have any permissions", ops.OperationName(op))
	}
	return perm, opd.PermissionRequired, nil
}

func (ops *operations[K]) Merge(nops Operations[K]) error {
	newOps, ok := nops.(*operations[K])
	if !ok {
		return ErrMergeInvalidOperations
	}

	for op, opd := range newOps.opDetails {
		ops.opDetails[op] = opd
		if opd.PermissionRequired {
			perm, exists := newOps.opPermission[op]
			if !exists {
				return fmt.Errorf("missing permission for required operation %s", opd.Name)
			}
			ops.opPermission[op] = perm
		}
	}

	return nil
}

func (ops *operations[K]) Remove(rops Operations[K]) error {
	remOps, ok := rops.(*operations[K])
	if !ok {
		return ErrRemoveInvalidOperations
	}

	for op := range remOps.opDetails {
		delete(ops.opDetails, op)
		delete(ops.opPermission, op)
	}

	return nil
}

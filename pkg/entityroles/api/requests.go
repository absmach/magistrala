// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
)

type createRoleReq struct {
	token           string
	entityID        string
	RoleName        string   `json:"role_name"`
	OptionalActions []string `json:"optional_actions"`
	OptionalMembers []string `json:"optional_members"`
}

func (req createRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if err := api.ValidateUUID(req.entityID); err != nil {
		return err
	}
	if len(req.RoleName) == 0 {
		return apiutil.ErrMissingRoleName
	}
	if len(req.RoleName) > 200 {
		return apiutil.ErrNameSize
	}

	return nil
}

type listRolesReq struct {
	token    string
	entityID string
	limit    uint64
	offset   uint64
}

func (req listRolesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	return nil
}

type viewRoleReq struct {
	token    string
	entityID string
	roleName string
}

func (req viewRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type updateRoleReq struct {
	token    string
	entityID string
	roleName string
	Name     string `json:"name"`
}

func (req updateRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" || req.Name == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type deleteRoleReq struct {
	token    string
	entityID string
	roleName string
}

func (req deleteRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type listAvailableActionsReq struct {
	token string
}

func (req listAvailableActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type addRoleActionsReq struct {
	token    string
	entityID string
	roleName string
	Actions  []string `json:"actions"`
}

func (req addRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}

	if len(req.Actions) == 0 {
		return apiutil.ErrMissingPolicyEntityType
	}
	return nil
}

type listRoleActionsReq struct {
	token    string
	entityID string
	roleName string
}

func (req listRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type deleteRoleActionsReq struct {
	token    string
	entityID string
	roleName string
	Actions  []string `json:"actions"`
}

func (req deleteRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}

	if len(req.Actions) == 0 {
		return apiutil.ErrMissingPolicyEntityType
	}
	return nil
}

type deleteAllRoleActionsReq struct {
	token    string
	entityID string
	roleName string
}

func (req deleteAllRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type addRoleMembersReq struct {
	token    string
	entityID string
	roleName string
	Members  []string `json:"members"`
}

func (req addRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	if len(req.Members) == 0 {
		return apiutil.ErrMissingRoleMembers
	}
	return nil
}

type listRoleMembersReq struct {
	token    string
	entityID string
	roleName string
	limit    uint64
	offset   uint64
}

func (req listRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	return nil
}

type deleteRoleMembersReq struct {
	token    string
	entityID string
	roleName string
	Members  []string `json:"members"`
}

func (req deleteRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	if len(req.Members) == 0 {
		return apiutil.ErrMissingRoleMembers
	}
	return nil
}

type deleteAllRoleMembersReq struct {
	token    string
	entityID string
	roleName string
}

func (req deleteAllRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleName == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

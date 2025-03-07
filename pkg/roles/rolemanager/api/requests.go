// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
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

type listEntityMembersReq struct {
	token            string
	entityID         string
	limit            uint64
	offset           uint64
	dir              string
	order            string
	accessProviderID string
	roleId           string
	roleName         string
	actions          []string
	accessType       string
}

func (req listEntityMembersReq) validate() error {
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

type removeEntityMembersReq struct {
	token     string
	entityID  string
	MemberIDs []string `json:"member_ids"`
}

func (req removeEntityMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if len(req.MemberIDs) == 0 {
		return apiutil.ErrMissingMemberIDs
	}
	return nil
}

type viewRoleReq struct {
	token    string
	entityID string
	roleID   string
}

func (req viewRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	return nil
}

type updateRoleReq struct {
	token    string
	entityID string
	roleID   string
	Name     string `json:"name"`
}

func (req updateRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	if req.Name == "" {
		return apiutil.ErrMissingRoleName
	}
	return nil
}

type deleteRoleReq struct {
	token    string
	entityID string
	roleID   string
}

func (req deleteRoleReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
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
	roleID   string
	Actions  []string `json:"actions"`
}

func (req addRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}

	if len(req.Actions) == 0 {
		return apiutil.ErrMissingPolicyEntityType
	}
	return nil
}

type listRoleActionsReq struct {
	token    string
	entityID string
	roleID   string
}

func (req listRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	return nil
}

type deleteRoleActionsReq struct {
	token    string
	entityID string
	roleID   string
	Actions  []string `json:"actions"`
}

func (req deleteRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}

	if len(req.Actions) == 0 {
		return apiutil.ErrMissingPolicyEntityType
	}
	return nil
}

type deleteAllRoleActionsReq struct {
	token    string
	entityID string
	roleID   string
}

func (req deleteAllRoleActionsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	return nil
}

type addRoleMembersReq struct {
	token    string
	entityID string
	roleID   string
	Members  []string `json:"members"`
}

func (req addRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	if len(req.Members) == 0 {
		return apiutil.ErrMissingRoleMembers
	}
	return nil
}

type listRoleMembersReq struct {
	token    string
	entityID string
	roleID   string
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
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	return nil
}

type deleteRoleMembersReq struct {
	token    string
	entityID string
	roleID   string
	Members  []string `json:"members"`
}

func (req deleteRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	if len(req.Members) == 0 {
		return apiutil.ErrMissingRoleMembers
	}
	return nil
}

type deleteAllRoleMembersReq struct {
	token    string
	entityID string
	roleID   string
}

func (req deleteAllRoleMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.entityID == "" {
		return apiutil.ErrMissingID
	}
	if req.roleID == "" {
		return apiutil.ErrMissingRoleID
	}
	return nil
}

// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/absmach/magistrala/pkg/errors"
)

func (sdk mgSDK) createRole(ctx context.Context, entityURL, entityEndpoint, id, workspaceID string, rq RoleReq, token string) (Role, errors.SDKError) {
	data, err := json.Marshal(rq)
	if err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Role{}, sdkErr
	}

	role := Role{}
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) listRoles(ctx context.Context, entityURL, entityEndpoint, id, workspaceID string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", workspaceID, entityEndpoint, id, rolesEndpoint)
	if entityEndpoint == workspacesEndpoint {
		endpoint = fmt.Sprintf("%s/%s/%s", entityEndpoint, id, rolesEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, endpoint, pm)
	if err != nil {
		return RolesPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return RolesPage{}, sdkErr
	}

	var rp RolesPage
	if err := json.Unmarshal(body, &rp); err != nil {
		return RolesPage{}, errors.NewSDKError(err)
	}

	return rp, nil
}

func (sdk mgSDK) viewRole(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID, token string) (Role, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Role{}, sdkErr
	}

	var role Role
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) updateRole(ctx context.Context, entityURL, entityEndpoint, id, roleID, newName, workspaceID string, token string) (Role, errors.SDKError) {
	ucr := updateRoleNameReq{Name: newName}
	data, err := json.Marshal(ucr)
	if err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Role{}, sdkErr
	}

	role := Role{}
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) deleteRole(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) addRoleActions(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, actions []string, token string) ([]string, errors.SDKError) {
	acra := roleActionsReq{Actions: actions}
	data, err := json.Marshal(acra)
	if err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return []string{}, sdkErr
	}

	res := roleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	return res.Actions, nil
}

func (sdk mgSDK) listRoleActions(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, token string) ([]string, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return nil, sdkErr
	}

	res := roleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.NewSDKError(err)
	}

	return res.Actions, nil
}

func (sdk mgSDK) removeRoleActions(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, actions []string, token string) errors.SDKError {
	rcra := roleActionsReq{Actions: actions}
	data, err := json.Marshal(rcra)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete")
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete")
	}
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) removeAllRoleActions(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete-all")
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete-all")
	}
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) addRoleMembers(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, members []string, token string) ([]string, errors.SDKError) {
	acrm := roleMembersReq{Members: members}
	data, err := json.Marshal(acrm)
	if err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return []string{}, sdkErr
	}

	res := roleMembersRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	return res.Members, nil
}

func (sdk mgSDK) listRoleMembers(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s/%s/%s", workspaceID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	if entityEndpoint == workspacesEndpoint {
		endpoint = fmt.Sprintf("%s/%s/%s/%s/%s", entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, endpoint, pm)
	if err != nil {
		return RoleMembersPage{}, errors.NewSDKError(err)
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return RoleMembersPage{}, sdkErr
	}

	res := RoleMembersPage{}
	if err := json.Unmarshal(body, &res); err != nil {
		return RoleMembersPage{}, errors.NewSDKError(err)
	}

	return res, nil
}

func (sdk mgSDK) removeRoleMembers(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID string, members []string, token string) errors.SDKError {
	rcrm := roleMembersReq{Members: members}
	data, err := json.Marshal(rcrm)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete")
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete")
	}
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) removeAllRoleMembers(ctx context.Context, entityURL, entityEndpoint, id, roleID, workspaceID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete-all")
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete-all")
	}
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) listAvailableRoleActions(ctx context.Context, entityURL, entityEndpoint, workspaceID, token string) ([]string, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, workspaceID, entityEndpoint, rolesEndpoint, "available-actions")
	if entityEndpoint == workspacesEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s", entityURL, entityEndpoint, rolesEndpoint, "available-actions")
	}
	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return nil, sdkErr
	}

	res := availableRoleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.NewSDKError(err)
	}

	return res.AvailableActions, nil
}

func (sdk mgSDK) listEntityMembers(ctx context.Context, entityURL, workspaceID, entityEndpoint, id, token string, pm PageMetadata) (EntityMembersPage, errors.SDKError) {
	ep := fmt.Sprintf("%s/%s/%s/%s/%s", workspaceID, entityEndpoint, id, rolesEndpoint, membersEndpoint)
	if entityEndpoint == workspacesEndpoint {
		ep = fmt.Sprintf("%s/%s/%s/%s", entityEndpoint, id, rolesEndpoint, membersEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, ep, pm)
	if err != nil {
		return EntityMembersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return EntityMembersPage{}, sdkErr
	}

	res := EntityMembersPage{}
	if err := json.Unmarshal(body, &res); err != nil {
		return EntityMembersPage{}, errors.NewSDKError(err)
	}

	return res, nil
}

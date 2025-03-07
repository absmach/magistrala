// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/absmach/supermq/pkg/errors"
)

func (sdk mgSDK) createRole(entityURL, entityEndpoint, id, domainID string, rq RoleReq, token string) (Role, errors.SDKError) {
	data, err := json.Marshal(rq)
	if err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Role{}, sdkerr
	}

	role := Role{}
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) listRoles(entityURL, entityEndpoint, id, domainID string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s", domainID, entityEndpoint, id, rolesEndpoint)
	if entityEndpoint == domainsEndpoint {
		endpoint = fmt.Sprintf("%s/%s/%s", entityEndpoint, id, rolesEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, endpoint, pm)
	if err != nil {
		return RolesPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return RolesPage{}, sdkerr
	}

	var rp RolesPage
	if err := json.Unmarshal(body, &rp); err != nil {
		return RolesPage{}, errors.NewSDKError(err)
	}

	return rp, nil
}

func (sdk mgSDK) viewRole(entityURL, entityEndpoint, id, roleID, domainID, token string) (Role, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Role{}, sdkerr
	}

	var role Role
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) updateRole(entityURL, entityEndpoint, id, roleID, newName, domainID string, token string) (Role, errors.SDKError) {
	ucr := updateRoleNameReq{Name: newName}
	data, err := json.Marshal(ucr)
	if err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Role{}, sdkerr
	}

	role := Role{}
	if err := json.Unmarshal(body, &role); err != nil {
		return Role{}, errors.NewSDKError(err)
	}

	return role, nil
}

func (sdk mgSDK) deleteRole(entityURL, entityEndpoint, id, roleID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID)
	}
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) addRoleActions(entityURL, entityEndpoint, id, roleID, domainID string, actions []string, token string) ([]string, errors.SDKError) {
	acra := roleActionsReq{Actions: actions}
	data, err := json.Marshal(acra)
	if err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return []string{}, sdkerr
	}

	res := roleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	return res.Actions, nil
}

func (sdk mgSDK) listRoleActions(entityURL, entityEndpoint, id, roleID, domainID string, token string) ([]string, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return nil, sdkerr
	}

	res := roleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.NewSDKError(err)
	}

	return res.Actions, nil
}

func (sdk mgSDK) removeRoleActions(entityURL, entityEndpoint, id, roleID, domainID string, actions []string, token string) errors.SDKError {
	rcra := roleActionsReq{Actions: actions}
	data, err := json.Marshal(rcra)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete")
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete")
	}
	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) removeAllRoleActions(entityURL, entityEndpoint, id, roleID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete-all")
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, actionsEndpoint, "delete-all")
	}
	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) addRoleMembers(entityURL, entityEndpoint, id, roleID, domainID string, members []string, token string) ([]string, errors.SDKError) {
	acrm := roleMembersReq{Members: members}
	data, err := json.Marshal(acrm)
	if err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return []string{}, sdkerr
	}

	res := roleMembersRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return []string{}, errors.NewSDKError(err)
	}

	return res.Members, nil
}

func (sdk mgSDK) listRoleMembers(entityURL, entityEndpoint, id, roleID, domainID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s/%s/%s/%s/%s", domainID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	if entityEndpoint == domainsEndpoint {
		endpoint = fmt.Sprintf("%s/%s/%s/%s/%s", entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, endpoint, pm)
	if err != nil {
		return RoleMembersPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return RoleMembersPage{}, sdkerr
	}

	res := RoleMembersPage{}
	if err := json.Unmarshal(body, &res); err != nil {
		return RoleMembersPage{}, errors.NewSDKError(err)
	}

	return res, nil
}

func (sdk mgSDK) removeRoleMembers(entityURL, entityEndpoint, id, roleID, domainID string, members []string, token string) errors.SDKError {
	rcrm := roleMembersReq{Members: members}
	data, err := json.Marshal(rcrm)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete")
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete")
	}
	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) removeAllRoleMembers(entityURL, entityEndpoint, id, roleID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete-all")
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", entityURL, entityEndpoint, id, rolesEndpoint, roleID, membersEndpoint, "delete-all")
	}
	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusNoContent)

	return sdkerr
}

func (sdk mgSDK) listAvailableRoleActions(entityURL, entityEndpoint, domainID, token string) ([]string, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", entityURL, domainID, entityEndpoint, rolesEndpoint, "available-actions")
	if entityEndpoint == domainsEndpoint {
		url = fmt.Sprintf("%s/%s/%s/%s", entityURL, entityEndpoint, rolesEndpoint, "available-actions")
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return nil, sdkerr
	}

	res := availableRoleActionsRes{}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, errors.NewSDKError(err)
	}

	return res.AvailableActions, nil
}

func (sdk mgSDK) listEntityMembers(entityURL, domainID, entityEndpoint, id, token string, pm PageMetadata) (EntityMembersPage, errors.SDKError) {
	ep := fmt.Sprintf("%s/%s/%s/%s/%s", domainID, entityEndpoint, id, rolesEndpoint, membersEndpoint)
	if entityEndpoint == domainsEndpoint {
		ep = fmt.Sprintf("%s/%s/%s/%s", entityEndpoint, id, rolesEndpoint, membersEndpoint)
	}
	url, err := sdk.withQueryParams(entityURL, ep, pm)
	if err != nil {
		return EntityMembersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return EntityMembersPage{}, sdkerr
	}

	res := EntityMembersPage{}
	if err := json.Unmarshal(body, &res); err != nil {
		return EntityMembersPage{}, errors.NewSDKError(err)
	}

	return res, nil
}

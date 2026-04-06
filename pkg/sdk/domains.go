// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/roles"
)

const (
	domainsEndpoint = "domains"
	freezeEndpoint  = "freeze"
)

// Domain represents magistrala domain.
type Domain struct {
	ID          string                    `json:"id,omitempty"`
	Name        string                    `json:"name,omitempty"`
	Metadata    Metadata                  `json:"metadata,omitempty"`
	Tags        []string                  `json:"tags,omitempty"`
	Route       string                    `json:"route,omitempty"`
	Status      string                    `json:"status,omitempty"`
	Permission  string                    `json:"permission,omitempty"`
	CreatedBy   string                    `json:"created_by,omitempty"`
	CreatedAt   time.Time                 `json:"created_at,omitempty"`
	UpdatedBy   string                    `json:"updated_by,omitempty"`
	UpdatedAt   time.Time                 `json:"updated_at,omitempty"`
	Permissions []string                  `json:"permissions,omitempty"`
	Roles       []roles.MemberRoleActions `json:"roles,omitempty"`
}

func (sdk mgSDK) CreateDomain(ctx context.Context, domain Domain, token string) (Domain, errors.SDKError) {
	data, err := json.Marshal(domain)
	if err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.domainsURL, domainsEndpoint)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkErr != nil {
		return Domain{}, sdkErr
	}

	var d Domain
	if err := json.Unmarshal(body, &d); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) Domains(ctx context.Context, pm PageMetadata, token string) (DomainsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.domainsURL, domainsEndpoint, pm)
	if err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return DomainsPage{}, sdkErr
	}

	var dp DomainsPage
	if err := json.Unmarshal(body, &dp); err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}

	return dp, nil
}

func (sdk mgSDK) Domain(ctx context.Context, domainID, token string) (Domain, errors.SDKError) {
	if domainID == "" {
		return Domain{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID)

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return Domain{}, sdkErr
	}

	var domain Domain
	if err := json.Unmarshal(body, &domain); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	return domain, nil
}

func (sdk mgSDK) UpdateDomain(ctx context.Context, domain Domain, token string) (Domain, errors.SDKError) {
	if domain.ID == "" {
		return Domain{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, domainsEndpoint, domain.ID)

	data, err := json.Marshal(domain)
	if err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkErr != nil {
		return Domain{}, sdkErr
	}

	var d Domain
	if err := json.Unmarshal(body, &d); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) EnableDomain(ctx context.Context, domainID, token string) errors.SDKError {
	return sdk.changeDomainStatus(ctx, token, domainID, enableEndpoint)
}

func (sdk mgSDK) DisableDomain(ctx context.Context, domainID, token string) errors.SDKError {
	return sdk.changeDomainStatus(ctx, token, domainID, disableEndpoint)
}

func (sdk mgSDK) FreezeDomain(ctx context.Context, domainID, token string) errors.SDKError {
	return sdk.changeDomainStatus(ctx, token, domainID, freezeEndpoint)
}

func (sdk mgSDK) changeDomainStatus(ctx context.Context, token, id, status string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, id, status)
	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	return sdkErr
}

func (sdk mgSDK) CreateDomainRole(ctx context.Context, id string, rq RoleReq, token string) (Role, errors.SDKError) {
	return sdk.createRole(ctx, sdk.domainsURL, domainsEndpoint, id, "", rq, token)
}

func (sdk mgSDK) DomainRoles(ctx context.Context, id string, pm PageMetadata, token string) (RolesPage, errors.SDKError) {
	return sdk.listRoles(ctx, sdk.domainsURL, domainsEndpoint, id, "", pm, token)
}

func (sdk mgSDK) DomainRole(ctx context.Context, id, roleID, token string) (Role, errors.SDKError) {
	return sdk.viewRole(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) UpdateDomainRole(ctx context.Context, id, roleID, newName string, token string) (Role, errors.SDKError) {
	return sdk.updateRole(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, newName, "", token)
}

func (sdk mgSDK) DeleteDomainRole(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.deleteRole(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AddDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleActions(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", actions, token)
}

func (sdk mgSDK) DomainRoleActions(ctx context.Context, id, roleID string, token string) ([]string, errors.SDKError) {
	return sdk.listRoleActions(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) RemoveDomainRoleActions(ctx context.Context, id, roleID string, actions []string, token string) errors.SDKError {
	return sdk.removeRoleActions(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", actions, token)
}

func (sdk mgSDK) RemoveAllDomainRoleActions(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.removeAllRoleActions(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AddDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) ([]string, errors.SDKError) {
	return sdk.addRoleMembers(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", members, token)
}

func (sdk mgSDK) DomainRoleMembers(ctx context.Context, id, roleID string, pm PageMetadata, token string) (RoleMembersPage, errors.SDKError) {
	return sdk.listRoleMembers(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", pm, token)
}

func (sdk mgSDK) RemoveDomainRoleMembers(ctx context.Context, id, roleID string, members []string, token string) errors.SDKError {
	return sdk.removeRoleMembers(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", members, token)
}

func (sdk mgSDK) RemoveAllDomainRoleMembers(ctx context.Context, id, roleID, token string) errors.SDKError {
	return sdk.removeAllRoleMembers(ctx, sdk.domainsURL, domainsEndpoint, id, roleID, "", token)
}

func (sdk mgSDK) AvailableDomainRoleActions(ctx context.Context, token string) ([]string, errors.SDKError) {
	return sdk.listAvailableRoleActions(ctx, sdk.domainsURL, domainsEndpoint, "", token)
}

func (sdk mgSDK) ListDomainMembers(ctx context.Context, domainID string, pm PageMetadata, token string) (EntityMembersPage, errors.SDKError) {
	return sdk.listEntityMembers(ctx, sdk.domainsURL, domainID, domainsEndpoint, domainID, token, pm)
}

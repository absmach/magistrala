// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

const domainsEndpoint = "domains"

// Domain represents magistrala domain.
type Domain struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Alias       string    `json:"alias,omitempty"`
	Status      string    `json:"status,omitempty"`
	Permission  string    `json:"permission,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedBy   string    `json:"updated_by,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
}

func (sdk mgSDK) CreateDomain(domain Domain, token string) (Domain, errors.SDKError) {
	data, err := json.Marshal(domain)
	if err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.domainsURL, domainsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Domain{}, sdkerr
	}

	var d Domain
	if err := json.Unmarshal(body, &d); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) UpdateDomain(domain Domain, token string) (Domain, errors.SDKError) {
	data, err := json.Marshal(domain)
	if err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, domainsEndpoint, domain.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Domain{}, sdkerr
	}

	var d Domain
	if err := json.Unmarshal(body, &d); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}
	return d, nil
}

func (sdk mgSDK) Domain(domainID, token string) (Domain, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Domain{}, sdkerr
	}

	var domain Domain
	if err := json.Unmarshal(body, &domain); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	return domain, nil
}

func (sdk mgSDK) DomainPermissions(domainID, token string) (Domain, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID, permissionsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Domain{}, sdkerr
	}

	var domain Domain
	if err := json.Unmarshal(body, &domain); err != nil {
		return Domain{}, errors.NewSDKError(err)
	}

	return domain, nil
}

func (sdk mgSDK) Domains(pm PageMetadata, token string) (DomainsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.domainsURL, domainsEndpoint, pm)
	if err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return DomainsPage{}, sdkerr
	}

	var dp DomainsPage
	if err := json.Unmarshal(body, &dp); err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}

	return dp, nil
}

func (sdk mgSDK) ListUserDomains(pm PageMetadata, token string) (DomainsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.domainsURL, domainsEndpoint, pm)
	if err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return DomainsPage{}, sdkerr
	}
	var dp DomainsPage
	if err := json.Unmarshal(body, &dp); err != nil {
		return DomainsPage{}, errors.NewSDKError(err)
	}

	return dp, nil
}

func (sdk mgSDK) EnableDomain(domainID, token string) errors.SDKError {
	return sdk.changeDomainStatus(token, domainID, enableEndpoint)
}

func (sdk mgSDK) DisableDomain(domainID, token string) errors.SDKError {
	return sdk.changeDomainStatus(token, domainID, disableEndpoint)
}

func (sdk mgSDK) changeDomainStatus(token, id, status string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, id, status)
	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) AddUserToDomain(domainID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID, usersEndpoint, assignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) RemoveUserFromDomain(domainID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID, usersEndpoint, unassignEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

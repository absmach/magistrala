// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	permissionsEndpoint = "permissions"
	clientsEndpoint     = "clients"
	connectEndpoint     = "connect"
	disconnectEndpoint  = "disconnect"
	identifyEndpoint    = "identify"
	shareEndpoint       = "share"
	unshareEndpoint     = "unshare"
)

// Client represents magistrala client.
type Client struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Credentials ClientCredentials      `json:"credentials"`
	Tags        []string               `json:"tags,omitempty"`
	DomainID    string                 `json:"domain_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
}

type ClientCredentials struct {
	Identity string `json:"identity,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

func (sdk mgSDK) CreateClient(client Client, domainID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(client)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	client = Client{}
	if err := json.Unmarshal(body, &client); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return client, nil
}

func (sdk mgSDK) CreateClients(clients []Client, domainID, token string) ([]Client, errors.SDKError) {
	data, err := json.Marshal(clients)
	if err != nil {
		return []Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, "bulk")

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return []Client{}, sdkerr
	}

	var ctr createClientsRes
	if err := json.Unmarshal(body, &ctr); err != nil {
		return []Client{}, errors.NewSDKError(err)
	}

	return ctr.Clients, nil
}

func (sdk mgSDK) Clients(pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, clientsEndpoint)
	url, err := sdk.withQueryParams(sdk.clientsURL, endpoint, pm)
	if err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ClientsPage{}, sdkerr
	}

	var cp ClientsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) ClientsByChannel(chanID string, pm PageMetadata, domainID, token string) (ClientsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.clientsURL, fmt.Sprintf("%s/channels/%s/%s", domainID, chanID, clientsEndpoint), pm)
	if err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ClientsPage{}, sdkerr
	}

	var tp ClientsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ClientsPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}

func (sdk mgSDK) Client(id, domainID, token string) (Client, errors.SDKError) {
	if id == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	var t Client
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) ClientPermissions(id, domainID, token string) (Client, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id, permissionsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	var t Client
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClient(t Client, domainID, token string) (Client, errors.SDKError) {
	if t.ID == "" {
		return Client{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, t.ID)

	data, err := json.Marshal(t)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	t = Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClientTags(t Client, domainID, token string) (Client, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/tags", sdk.clientsURL, domainID, clientsEndpoint, t.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	t = Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) UpdateClientSecret(id, secret, domainID, token string) (Client, errors.SDKError) {
	ucsr := updateClientSecretReq{Secret: secret}

	data, err := json.Marshal(ucsr)
	if err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/secret", sdk.clientsURL, domainID, clientsEndpoint, id)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	var t Client
	if err = json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) EnableClient(id, domainID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(id, enableEndpoint, domainID, token)
}

func (sdk mgSDK) DisableClient(id, domainID, token string) (Client, errors.SDKError) {
	return sdk.changeClientStatus(id, disableEndpoint, domainID, token)
}

func (sdk mgSDK) changeClientStatus(id, status, domainID, token string) (Client, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id, status)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Client{}, sdkerr
	}

	t := Client{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Client{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mgSDK) ShareClient(clientID string, req UsersRelationRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, clientID, shareEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mgSDK) UnshareClient(clientID string, req UsersRelationRequest, domainID, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, clientID, unshareEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) ListClientUsers(clientID string, pm PageMetadata, domainID, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s/%s", domainID, clientsEndpoint, clientID, usersEndpoint), pm)
	if err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}
	up := UsersPage{}
	if err := json.Unmarshal(body, &up); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return up, nil
}

func (sdk mgSDK) DeleteClient(id, domainID, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.clientsURL, domainID, clientsEndpoint, id)
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

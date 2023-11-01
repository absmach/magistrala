// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

const (
	thingsEndpoint     = "things"
	connectEndpoint    = "connect"
	disconnectEndpoint = "disconnect"
	identifyEndpoint   = "identify"
	shareEndpoint      = "share"
	unshareEndpoint    = "unshare"
)

// Thing represents mainflux thing.
type Thing struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	Credentials Credentials            `json:"credentials"`
	Tags        []string               `json:"tags,omitempty"`
	Owner       string                 `json:"owner,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
	Status      string                 `json:"status,omitempty"`
}

func (sdk mfSDK) CreateThing(thing Thing, token string) (Thing, errors.SDKError) {
	data, err := json.Marshal(thing)
	if err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.thingsURL, thingsEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	thing = Thing{}
	if err := json.Unmarshal(body, &thing); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return thing, nil
}

func (sdk mfSDK) CreateThings(things []Thing, token string) ([]Thing, errors.SDKError) {
	data, err := json.Marshal(things)
	if err != nil {
		return []Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, "bulk")

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return []Thing{}, sdkerr
	}

	var ctr createThingsRes
	if err := json.Unmarshal(body, &ctr); err != nil {
		return []Thing{}, errors.NewSDKError(err)
	}

	return ctr.Things, nil
}

func (sdk mfSDK) Things(pm PageMetadata, token string) (ThingsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, thingsEndpoint, pm)
	if err != nil {
		return ThingsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ThingsPage{}, sdkerr
	}

	var cp ThingsPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return ThingsPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mfSDK) ThingsByChannel(chanID string, pm PageMetadata, token string) (ThingsPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.thingsURL, fmt.Sprintf("channels/%s/%s", chanID, thingsEndpoint), pm)
	if err != nil {
		return ThingsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return ThingsPage{}, sdkerr
	}

	var tp ThingsPage
	if err := json.Unmarshal(body, &tp); err != nil {
		return ThingsPage{}, errors.NewSDKError(err)
	}

	return tp, nil
}

func (sdk mfSDK) Thing(id, token string) (Thing, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, id)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	var t Thing
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) UpdateThing(t Thing, token string) (Thing, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.thingsURL, thingsEndpoint, t.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	t = Thing{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) UpdateThingTags(t Thing, token string) (Thing, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/tags", sdk.thingsURL, thingsEndpoint, t.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	t = Thing{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) UpdateThingSecret(id, secret, token string) (Thing, errors.SDKError) {
	ucsr := updateThingSecretReq{Secret: secret}

	data, err := json.Marshal(ucsr)
	if err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/secret", sdk.thingsURL, thingsEndpoint, id)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	var t Thing
	if err = json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) UpdateThingOwner(t Thing, token string) (Thing, errors.SDKError) {
	data, err := json.Marshal(t)
	if err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/owner", sdk.thingsURL, thingsEndpoint, t.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	t = Thing{}
	if err = json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) EnableThing(id, token string) (Thing, errors.SDKError) {
	return sdk.changeThingStatus(id, enableEndpoint, token)
}

func (sdk mfSDK) DisableThing(id, token string) (Thing, errors.SDKError) {
	return sdk.changeThingStatus(id, disableEndpoint, token)
}

func (sdk mfSDK) changeThingStatus(id, status, token string) (Thing, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, thingsEndpoint, id, status)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Thing{}, sdkerr
	}

	t := Thing{}
	if err := json.Unmarshal(body, &t); err != nil {
		return Thing{}, errors.NewSDKError(err)
	}

	return t, nil
}

func (sdk mfSDK) IdentifyThing(key string) (string, errors.SDKError) {
	url := fmt.Sprintf("%s/%s", sdk.thingsURL, identifyEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, ThingPrefix+key, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return "", sdkerr
	}

	var i identifyThingResp
	if err := json.Unmarshal(body, &i); err != nil {
		return "", errors.NewSDKError(err)
	}

	return i.ID, nil
}

func (sdk mfSDK) ShareThing(thingID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, thingsEndpoint, thingID, shareEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	return sdkerr
}

func (sdk mfSDK) UnshareThing(thingID string, req UsersRelationRequest, token string) errors.SDKError {
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.thingsURL, thingsEndpoint, thingID, unshareEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mfSDK) ListThingUsers(thingID string, pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s/%s", thingsEndpoint, thingID, usersEndpoint), pm)
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

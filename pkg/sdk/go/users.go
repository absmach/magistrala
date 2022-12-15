// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	usersEndpoint    = "users"
	tokensEndpoint   = "tokens"
	passwordEndpoint = "password"
	membersEndpoint  = "members"
)

func (sdk mfSDK) CreateUser(token string, u User) (string, errors.SDKError) {
	data, err := json.Marshal(u)
	if err != nil {
		return "", errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s", sdk.usersURL, usersEndpoint)

	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), fmt.Sprintf("/%s/", usersEndpoint))
	return id, nil
}

func (sdk mfSDK) User(userID, token string) (User, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, userID)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return User{}, err
	}

	var u User
	if err := json.Unmarshal(body, &u); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return u, nil
}

func (sdk mfSDK) Users(token string, pm PageMetadata) (UsersPage, errors.SDKError) {
	var url string
	var err error

	if url, err = sdk.withQueryParams(sdk.usersURL, usersEndpoint, pm); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}

	var up UsersPage
	if err := json.Unmarshal(body, &up); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return up, nil
}

func (sdk mfSDK) CreateToken(user User) (string, errors.SDKError) {
	data, err := json.Marshal(user)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, tokensEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, "", string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	var tr tokenRes
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", errors.NewSDKError(err)
	}

	return tr.Token, nil
}

func (sdk mfSDK) UpdateUser(u User, token string) errors.SDKError {
	data, err := json.Marshal(u)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, usersEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, token, string(CTJSON), data, http.StatusOK)
	return sdkerr
}

func (sdk mfSDK) UpdatePassword(oldPass, newPass, token string) errors.SDKError {
	ur := UserPasswordReq{
		OldPassword: oldPass,
		Password:    newPass,
	}
	data, err := json.Marshal(ur)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, passwordEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPatch, url, token, string(CTJSON), data, http.StatusCreated)
	return sdkerr
}

func (sdk mfSDK) EnableUser(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/enable", sdk.usersURL, usersEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

func (sdk mfSDK) DisableUser(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/disable", sdk.usersURL, usersEndpoint, id)
	_, _, err := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

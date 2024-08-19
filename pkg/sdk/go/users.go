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
	usersEndpoint         = "users"
	assignEndpoint        = "assign"
	unassignEndpoint      = "unassign"
	enableEndpoint        = "enable"
	disableEndpoint       = "disable"
	issueTokenEndpoint    = "tokens/issue"
	refreshTokenEndpoint  = "tokens/refresh"
	membersEndpoint       = "members"
	PasswordResetEndpoint = "password"
)

// User represents magistrala user its credentials.
type User struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Credentials Credentials `json:"credentials"`
	Tags        []string    `json:"tags,omitempty"`
	Domain      string      `json:"-"` // ignoring Domain Field, since it will be always empty for users
	Metadata    Metadata    `json:"metadata,omitempty"`
	CreatedAt   time.Time   `json:"created_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at,omitempty"`
	Status      string      `json:"status,omitempty"`
	Role        string      `json:"role,omitempty"`
}

func (sdk mgSDK) CreateUser(user User, token string) (User, errors.SDKError) {
	data, err := json.Marshal(user)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, usersEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user = User{}
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) Users(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, usersEndpoint, pm)
	if err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}

	var cp UsersPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) User(id, token string) (User, errors.SDKError) {
	if id == "" {
		return User{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, id)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) UserProfile(token string) (User, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/profile", sdk.usersURL, usersEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) UpdateUser(user User, token string) (User, errors.SDKError) {
	if user.ID == "" {
		return User{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user = User{}
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) UpdateUserTags(user User, token string) (User, errors.SDKError) {
	data, err := json.Marshal(user)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/tags", sdk.usersURL, usersEndpoint, user.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user = User{}
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) UpdateUserIdentity(user User, token string) (User, errors.SDKError) {
	ucir := updateClientIdentityReq{token: token, id: user.ID, Identity: user.Credentials.Identity}

	data, err := json.Marshal(ucir)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/identity", sdk.usersURL, usersEndpoint, user.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user = User{}
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) ResetPasswordRequest(email string) errors.SDKError {
	rpr := resetPasswordRequestreq{Email: email}

	data, err := json.Marshal(rpr)
	if err != nil {
		return errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s/reset-request", sdk.usersURL, PasswordResetEndpoint)

	header := make(map[string]string)
	header["Referer"] = sdk.HostURL

	_, _, sdkerr := sdk.processRequest(http.MethodPost, url, "", data, header, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) ResetPassword(password, confPass, token string) errors.SDKError {
	rpr := resetPasswordReq{Token: token, Password: password, ConfPass: confPass}

	data, err := json.Marshal(rpr)
	if err != nil {
		return errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s/reset", sdk.usersURL, PasswordResetEndpoint)

	_, _, sdkerr := sdk.processRequest(http.MethodPut, url, "", data, nil, http.StatusCreated)

	return sdkerr
}

func (sdk mgSDK) UpdatePassword(oldPass, newPass, token string) (User, errors.SDKError) {
	ucsr := updateClientSecretReq{OldSecret: oldPass, NewSecret: newPass}

	data, err := json.Marshal(ucsr)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/secret", sdk.usersURL, usersEndpoint)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	var user User
	if err = json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) UpdateUserRole(user User, token string) (User, errors.SDKError) {
	data, err := json.Marshal(user)
	if err != nil {
		return User{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/role", sdk.usersURL, usersEndpoint, user.ID)

	_, body, sdkerr := sdk.processRequest(http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user = User{}
	if err = json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) ListChannelUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, usersEndpoint, pm)
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

func (sdk mgSDK) ListGroupUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s", usersEndpoint, membersEndpoint), pm)
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

func (sdk mgSDK) ListThingUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s", usersEndpoint, membersEndpoint), pm)
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

func (sdk mgSDK) ListDomainUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/%s", usersEndpoint, membersEndpoint), pm)
	if err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}
	var up UsersPage
	if err := json.Unmarshal(body, &up); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return up, nil
}

func (sdk mgSDK) SearchUsers(pm PageMetadata, token string) (UsersPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, fmt.Sprintf("%s/search", usersEndpoint), pm)
	if err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return UsersPage{}, sdkerr
	}

	var cp UsersPage
	if err := json.Unmarshal(body, &cp); err != nil {
		return UsersPage{}, errors.NewSDKError(err)
	}

	return cp, nil
}

func (sdk mgSDK) EnableUser(id, token string) (User, errors.SDKError) {
	return sdk.changeClientStatus(token, id, enableEndpoint)
}

func (sdk mgSDK) DisableUser(id, token string) (User, errors.SDKError) {
	return sdk.changeClientStatus(token, id, disableEndpoint)
}

func (sdk mgSDK) changeClientStatus(token, id, status string) (User, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.usersURL, usersEndpoint, id, status)

	_, body, sdkerr := sdk.processRequest(http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return User{}, sdkerr
	}

	user := User{}
	if err := json.Unmarshal(body, &user); err != nil {
		return User{}, errors.NewSDKError(err)
	}

	return user, nil
}

func (sdk mgSDK) DeleteUser(id, token string) errors.SDKError {
	if id == "" {
		return errors.NewSDKError(apiutil.ErrMissingID)
	}
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, usersEndpoint, id)
	_, _, sdkerr := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

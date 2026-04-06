// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

const (
	invitationsEndpoint = "invitations"
	acceptEndpoint      = "accept"
	rejectEndpoint      = "reject"
)

type Invitation struct {
	InvitedBy     string    `json:"invited_by"`
	InviteeUserID string    `json:"invitee_user_id"`
	DomainID      string    `json:"domain_id"`
	DomainName    string    `json:"domain_name,omitempty"`
	RoleID        string    `json:"role_id,omitempty"`
	RoleName      string    `json:"role_name,omitempty"`
	Actions       []string  `json:"actions,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	ConfirmedAt   time.Time `json:"confirmed_at,omitempty"`
	RejectedAt    time.Time `json:"rejected_at,omitempty"`
	Resend        bool      `json:"resend,omitempty"`
}

type InvitationPage struct {
	Total       uint64       `json:"total"`
	Offset      uint64       `json:"offset"`
	Limit       uint64       `json:"limit"`
	Invitations []Invitation `json:"invitations"`
}

func (sdk mgSDK) SendInvitation(ctx context.Context, invitation Invitation, token string) (err error) {
	data, err := json.Marshal(invitation)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, invitation.DomainID, invitationsEndpoint)

	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated)

	return sdkErr
}

func (sdk mgSDK) DomainInvitations(ctx context.Context, pm PageMetadata, token, domainID string) (invitations InvitationPage, err error) {
	url := fmt.Sprintf("%s/%s/%s", domainsEndpoint, domainID, invitationsEndpoint)
	url, err = sdk.withQueryParams(sdk.domainsURL, url, pm)
	if err != nil {
		return InvitationPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return InvitationPage{}, sdkerr
	}

	var invPage InvitationPage
	if err := json.Unmarshal(body, &invPage); err != nil {
		return InvitationPage{}, errors.NewSDKError(err)
	}

	return invPage, nil
}

func (sdk mgSDK) Invitations(ctx context.Context, pm PageMetadata, token string) (invitations InvitationPage, err error) {
	url, err := sdk.withQueryParams(sdk.domainsURL, invitationsEndpoint, pm)
	if err != nil {
		return InvitationPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return InvitationPage{}, sdkErr
	}

	var invPage InvitationPage
	if err := json.Unmarshal(body, &invPage); err != nil {
		return InvitationPage{}, errors.NewSDKError(err)
	}

	return invPage, nil
}

func (sdk mgSDK) AcceptInvitation(ctx context.Context, domainID, token string) (err error) {
	req := struct {
		DomainID string `json:"domain_id"`
	}{
		DomainID: domainID,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, invitationsEndpoint, acceptEndpoint)

	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) RejectInvitation(ctx context.Context, domainID, token string) (err error) {
	req := struct {
		DomainID string `json:"domain_id"`
	}{
		DomainID: domainID,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.domainsURL, invitationsEndpoint, rejectEndpoint)

	_, _, sdkErr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

func (sdk mgSDK) DeleteInvitation(ctx context.Context, userID, domainID, token string) (err error) {
	req := struct {
		UserID string `json:"user_id"`
	}{
		UserID: userID,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.domainsURL, domainsEndpoint, domainID, invitationsEndpoint)

	_, _, sdkErr := sdk.processRequest(ctx, http.MethodDelete, url, token, data, nil, http.StatusNoContent)

	return sdkErr
}

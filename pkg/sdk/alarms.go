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

const alarmsEndpoint = "alarms"

// Alarm represents an alarm instance.
type Alarm struct {
	ID             string    `json:"id,omitempty"`
	RuleID         string    `json:"rule_id,omitempty"`
	DomainID       string    `json:"domain_id,omitempty"`
	ChannelID      string    `json:"channel_id,omitempty"`
	ClientID       string    `json:"client_id,omitempty"`
	Subtopic       string    `json:"subtopic,omitempty"`
	Status         string    `json:"status,omitempty"`
	Measurement    string    `json:"measurement,omitempty"`
	Value          string    `json:"value,omitempty"`
	Unit           string    `json:"unit,omitempty"`
	Threshold      string    `json:"threshold,omitempty"`
	Cause          string    `json:"cause,omitempty"`
	Severity       uint8     `json:"severity,omitempty"`
	AssigneeID     string    `json:"assignee_id,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	UpdatedBy      string    `json:"updated_by,omitempty"`
	AssignedAt     time.Time `json:"assigned_at,omitempty"`
	AssignedBy     string    `json:"assigned_by,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string    `json:"acknowledged_by,omitempty"`
	ResolvedAt     time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     string    `json:"resolved_by,omitempty"`
	Metadata       Metadata  `json:"metadata,omitempty"`
}

type AlarmsPage struct {
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Total  uint64  `json:"total"`
	Alarms []Alarm `json:"alarms"`
}

func (sdk mgSDK) UpdateAlarm(ctx context.Context, alarm Alarm, domainID, token string) (Alarm, errors.SDKError) {
	data, err := json.Marshal(alarm)
	if err != nil {
		return Alarm{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.alarmsURL, domainID, alarmsEndpoint, alarm.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPut, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Alarm{}, sdkerr
	}

	var a Alarm
	if err := json.Unmarshal(body, &a); err != nil {
		return Alarm{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) ViewAlarm(ctx context.Context, id, domainID, token string) (Alarm, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.alarmsURL, domainID, alarmsEndpoint, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Alarm{}, sdkerr
	}

	var a Alarm
	if err := json.Unmarshal(body, &a); err != nil {
		return Alarm{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) ListAlarms(ctx context.Context, pm PageMetadata, domainID, token string) (AlarmsPage, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, alarmsEndpoint)
	url, err := sdk.withQueryParams(sdk.alarmsURL, endpoint, pm)
	if err != nil {
		return AlarmsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return AlarmsPage{}, sdkerr
	}

	var ap AlarmsPage
	if err := json.Unmarshal(body, &ap); err != nil {
		return AlarmsPage{}, errors.NewSDKError(err)
	}

	return ap, nil
}

func (sdk mgSDK) DeleteAlarm(ctx context.Context, id, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.alarmsURL, domainID, alarmsEndpoint, id)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent, http.StatusOK)
	return sdkerr
}

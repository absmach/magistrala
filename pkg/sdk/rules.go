// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/absmach/supermq/pkg/errors"
)

const rulesEndpoint = "rules"

// Rule represents a rule configuration.
type Rule struct {
	ID           string   `json:"id,omitempty"`
	Name         string   `json:"name,omitempty"`
	DomainID     string   `json:"domain,omitempty"`
	Metadata     Metadata `json:"metadata,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	InputChannel string   `json:"input_channel,omitempty"`
	InputTopic   string   `json:"input_topic,omitempty"`
	Logic        any      `json:"logic,omitempty"`
	Outputs      any      `json:"outputs,omitempty"`
	Schedule     any      `json:"schedule,omitempty"`
	Status       string   `json:"status,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	CreatedBy    string   `json:"created_by,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	UpdatedBy    string   `json:"updated_by,omitempty"`
}

type Page struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Total  uint64 `json:"total"`
	Rules  []Rule `json:"rules"`
}

func (sdk mgSDK) AddRule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError) {
	data, err := json.Marshal(r)
	if err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s", sdk.rulesEngineURL, domainID, rulesEndpoint)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, data, nil, http.StatusCreated, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) ViewRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.rulesEngineURL, domainID, rulesEndpoint, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) UpdateRule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError) {
	data, err := json.Marshal(r)
	if err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", sdk.rulesEngineURL, domainID, rulesEndpoint, r.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) UpdateRuleTags(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError) {
	data, err := json.Marshal(map[string]any{"tags": r.Tags})
	if err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/tags", sdk.rulesEngineURL, domainID, rulesEndpoint, r.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) UpdateRuleSchedule(ctx context.Context, r Rule, domainID, token string) (Rule, errors.SDKError) {
	data, err := json.Marshal(map[string]any{"schedule": r.Schedule})
	if err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s/%s/%s/schedule", sdk.rulesEngineURL, domainID, rulesEndpoint, r.ID)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, data, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) ListRules(ctx context.Context, pm PageMetadata, domainID, token string) (Page, errors.SDKError) {
	endpoint := fmt.Sprintf("%s/%s", domainID, rulesEndpoint)
	url, err := sdk.withQueryParams(sdk.rulesEngineURL, endpoint, pm)
	if err != nil {
		return Page{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Page{}, sdkerr
	}

	var ap Page
	if err := json.Unmarshal(body, &ap); err != nil {
		return Page{}, errors.NewSDKError(err)
	}

	return ap, nil
}

func (sdk mgSDK) RemoveRule(ctx context.Context, id, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.rulesEngineURL, domainID, rulesEndpoint, id)

	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent, http.StatusOK)
	return sdkerr
}

func (sdk mgSDK) EnableRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/enable", sdk.rulesEngineURL, domainID, rulesEndpoint, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

func (sdk mgSDK) DisableRule(ctx context.Context, id, domainID, token string) (Rule, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/disable", sdk.rulesEngineURL, domainID, rulesEndpoint, id)

	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Rule{}, sdkerr
	}

	var a Rule
	if err := json.Unmarshal(body, &a); err != nil {
		return Rule{}, errors.NewSDKError(err)
	}

	return a, nil
}

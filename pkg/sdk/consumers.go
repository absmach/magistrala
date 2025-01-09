// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/absmach/supermq/pkg/errors"
)

const (
	subscriptionEndpoint = "subscriptions"
)

type Subscription struct {
	ID      string `json:"id,omitempty"`
	OwnerID string `json:"owner_id,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Contact string `json:"contact,omitempty"`
}

func (sdk mgSDK) CreateSubscription(topic, contact, token string) (string, errors.SDKError) {
	sub := Subscription{
		Topic:   topic,
		Contact: contact,
	}
	data, err := json.Marshal(sub)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, subscriptionEndpoint)

	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, data, nil, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), fmt.Sprintf("/%s/", subscriptionEndpoint))

	return id, nil
}

func (sdk mgSDK) ListSubscriptions(pm PageMetadata, token string) (SubscriptionPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.usersURL, subscriptionEndpoint, pm)
	if err != nil {
		return SubscriptionPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return SubscriptionPage{}, sdkerr
	}

	var sp SubscriptionPage
	if err := json.Unmarshal(body, &sp); err != nil {
		return SubscriptionPage{}, errors.NewSDKError(err)
	}

	return sp, nil
}

func (sdk mgSDK) ViewSubscription(id, token string) (Subscription, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, subscriptionEndpoint, id)

	_, body, err := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if err != nil {
		return Subscription{}, err
	}

	var sub Subscription
	if err := json.Unmarshal(body, &sub); err != nil {
		return Subscription{}, errors.NewSDKError(err)
	}

	return sub, nil
}

func (sdk mgSDK) DeleteSubscription(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, subscriptionEndpoint, id)

	_, _, err := sdk.processRequest(http.MethodDelete, url, token, nil, nil, http.StatusNoContent)

	return err
}

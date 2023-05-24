package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"
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

func (sdk mfSDK) CreateSubscription(topic, contact, token string) (string, errors.SDKError) {
	sub := Subscription{
		Topic:   topic,
		Contact: contact,
	}
	data, err := json.Marshal(sub)
	if err != nil {
		return "", errors.NewSDKError(err)
	}

	url := fmt.Sprintf("%s/%s", sdk.usersURL, subscriptionEndpoint)
	headers, _, sdkerr := sdk.processRequest(http.MethodPost, url, token, string(CTJSON), data, http.StatusCreated)
	if sdkerr != nil {
		return "", sdkerr
	}

	id := strings.TrimPrefix(headers.Get("Location"), fmt.Sprintf("/%s/", subscriptionEndpoint))
	return id, nil
}

func (sdk mfSDK) ListSubscriptions(pm PageMetadata, token string) (SubscriptionPage, errors.SDKError) {
	url, err := sdk.withQueryParams(sdk.certsURL, subscriptionEndpoint, pm)
	if err != nil {
		return SubscriptionPage{}, errors.NewSDKError(err)
	}
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return SubscriptionPage{}, errors.NewSDKError(err)
	}

	var sp SubscriptionPage
	if err := json.Unmarshal(body, &sp); err != nil {
		return SubscriptionPage{}, errors.NewSDKError(err)
	}

	return sp, nil
}

func (sdk mfSDK) ViewSubscription(id, token string) (Subscription, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, subscriptionEndpoint, id)
	_, body, err := sdk.processRequest(http.MethodGet, url, token, string(CTJSON), nil, http.StatusOK)
	if err != nil {
		return Subscription{}, err
	}

	var sub Subscription
	if err := json.Unmarshal(body, &sub); err != nil {
		return Subscription{}, errors.NewSDKError(err)
	}

	return sub, nil
}

func (sdk mfSDK) DeleteSubscription(id, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s", sdk.usersURL, subscriptionEndpoint, id)

	_, _, err := sdk.processRequest(http.MethodDelete, url, token, string(CTJSON), nil, http.StatusNoContent)
	return err
}

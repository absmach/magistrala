// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

const eventsEndpoint = "events"

type Event struct {
	ID         string    `json:"id,omitempty"`
	Operation  string    `json:"operation,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
	Payload    Metadata  `json:"payload,omitempty"`
}

type EventsPage struct {
	Total  uint64  `json:"total"`
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Events []Event `json:"events"`
}

func (sdk mgSDK) Events(pm PageMetadata, id, entityType, token string) (events EventsPage, err error) {
	url, err := sdk.withQueryParams(sdk.eventsURL, eventsEndpoint+"/"+id+"/"+entityType, pm)
	if err != nil {
		return EventsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return EventsPage{}, sdkerr
	}

	var eventsPage EventsPage
	if err := json.Unmarshal(body, &eventsPage); err != nil {
		return EventsPage{}, errors.NewSDKError(err)
	}

	return eventsPage, nil
}

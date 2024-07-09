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

const journalEndpoint = "journal"

type Journal struct {
	ID         string    `json:"id,omitempty"`
	Operation  string    `json:"operation,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
	Attributes Metadata  `json:"attributes,omitempty"`
	Metadata   Metadata  `json:"metadata,omitempty"`
}

type JournalsPage struct {
	Total    uint64    `json:"total"`
	Offset   uint64    `json:"offset"`
	Limit    uint64    `json:"limit"`
	Journals []Journal `json:"journals"`
}

func (sdk mgSDK) Journal(entityType, entityID string, pm PageMetadata, token string) (journals JournalsPage, err error) {
	if entityID == "" {
		return JournalsPage{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	if entityType == "" {
		return JournalsPage{}, errors.NewSDKError(apiutil.ErrMissingEntityType)
	}

	url, err := sdk.withQueryParams(sdk.journalURL, fmt.Sprintf("%s/%s/%s", journalEndpoint, entityType, entityID), pm)
	if err != nil {
		return JournalsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkerr := sdk.processRequest(http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return JournalsPage{}, sdkerr
	}

	var journalsPage JournalsPage
	if err := json.Unmarshal(body, &journalsPage); err != nil {
		return JournalsPage{}, errors.NewSDKError(err)
	}

	return journalsPage, nil
}

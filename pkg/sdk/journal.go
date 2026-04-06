// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
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

func (sdk mgSDK) Journal(ctx context.Context, entityType, entityID, domainID string, pm PageMetadata, token string) (journals JournalsPage, err error) {
	if entityID == "" {
		return JournalsPage{}, errors.NewSDKError(apiutil.ErrMissingID)
	}
	if entityType == "" {
		return JournalsPage{}, errors.NewSDKError(apiutil.ErrMissingEntityType)
	}

	reqUrl := fmt.Sprintf("%s/%s/%s/%s", domainID, journalEndpoint, entityType, entityID)
	if entityType == "user" {
		reqUrl = fmt.Sprintf("%s/%s/%s", journalEndpoint, entityType, entityID)
	}

	url, err := sdk.withQueryParams(sdk.journalURL, reqUrl, pm)
	if err != nil {
		return JournalsPage{}, errors.NewSDKError(err)
	}

	_, body, sdkErr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkErr != nil {
		return JournalsPage{}, sdkErr
	}

	var journalsPage JournalsPage
	if err := json.Unmarshal(body, &journalsPage); err != nil {
		return JournalsPage{}, errors.NewSDKError(err)
	}

	return journalsPage, nil
}

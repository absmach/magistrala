// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/journal/api"
	"github.com/absmach/magistrala/journal/mocks"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupJournal() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)

	logger := mglog.NewMock()
	mux := api.MakeHandler(svc, logger, "journal-log", "test")
	return httptest.NewServer(mux), svc
}

func TestRetrieveJournal(t *testing.T) {
	js, svc := setupJournal()
	defer js.Close()

	testJournal := generateTestJournal(t)
	validEntityType := "user"

	sdkConf := sdk.Config{
		JournalURL: js.URL,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc       string
		token      string
		entityType string
		entityID   string
		pageMeta   sdk.PageMetadata
		svcReq     journal.Page
		svcRes     journal.JournalsPage
		svcErr     error
		response   sdk.JournalsPage
		err        error
	}{
		{
			desc:       "retrieve journal successfully",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.UserEntity,
				Direction:  "desc",
			},
			svcRes: journal.JournalsPage{
				Total:    1,
				Journals: []journal.Journal{convertJournal(testJournal)},
			},
			svcErr: nil,
			response: sdk.JournalsPage{
				Total:    1,
				Journals: []sdk.Journal{testJournal},
			},
			err: nil,
		},
		{
			desc:       "retrieve journal with invalid token",
			token:      invalidToken,
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.UserEntity,
				Direction:  "desc",
			},
			svcRes:   journal.JournalsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:       "retrieve journal with empty token",
			token:      "",
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:       "retrieve journal with invalid entity type",
			token:      validToken,
			entityType: "invalid",
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidEntityType), http.StatusBadRequest),
		},
		{
			desc:       "retrieve journal with empty entity ID",
			token:      validToken,
			entityType: validEntityType,
			entityID:   "",
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:       "retrieve journal with empty entity type",
			token:      validToken,
			entityType: "",
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(apiutil.ErrMissingEntityType),
		},
		{
			desc:       "retrieve journal with limit greater than default",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  1000,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:       "retrieve journal with invalid page metadata",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:       "retrieve journal with response that cannot be unmarshalled",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.UserEntity,
				Direction:  "desc",
			},
			svcRes: journal.JournalsPage{
				Total: 1,
				Journals: []journal.Journal{{
					ID:         validID,
					Operation:  "create",
					OccurredAt: time.Now(),
					Attributes: validMetadata,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveAll", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Journal(tc.entityType, tc.entityID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func generateTestJournal(t *testing.T) sdk.Journal {
	occuredAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return sdk.Journal{
		ID:         validID,
		Operation:  "create",
		OccurredAt: occuredAt,
		Attributes: validMetadata,
		Metadata:   validMetadata,
	}
}

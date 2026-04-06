// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/journal/api"
	"github.com/absmach/magistrala/journal/mocks"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupJournal() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	authn := new(authnmocks.Authentication)
	logger := mglog.NewMock()
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	mux := api.MakeHandler(svc, am, logger, "journal-log", "test")

	return httptest.NewServer(mux), svc, authn
}

func TestRetrieveJournal(t *testing.T) {
	js, svc, authn := setupJournal()
	defer js.Close()

	testJournal := generateTestJournal(t)
	validEntityType := "group"

	sdkConf := sdk.Config{
		JournalURL: js.URL,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc       string
		token      string
		session    smqauthn.Session
		entityType string
		entityID   string
		domainID   string
		pageMeta   sdk.PageMetadata
		svcReq     journal.Page
		svcRes     journal.JournalsPage
		svcErr     error
		authnErr   error
		response   sdk.JournalsPage
		err        error
	}{
		{
			desc:       "retrieve user journal successfully",
			token:      validToken,
			entityType: "user",
			entityID:   validID,
			domainID:   domainID,
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
			desc:       "retrieve channel journal successfully",
			token:      validToken,
			entityType: "channel",
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.ChannelEntity,
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
			desc:       "retrieve group journal successfully",
			token:      validToken,
			entityType: "group",
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.GroupEntity,
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
			desc:       "retrieve client journal successfully",
			token:      validToken,
			entityType: "client",
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.ClientEntity,
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
				EntityType: journal.GroupEntity,
				Direction:  "desc",
			},
			svcRes:   journal.JournalsPage{},
			authnErr: svcerr.ErrAuthentication,
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
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:       "retrieve journal with invalid entity type",
			token:      validToken,
			entityType: "invalid",
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrInvalidEntityType, http.StatusBadRequest),
		},
		{
			desc:       "retrieve journal with empty entity ID",
			token:      validToken,
			entityType: validEntityType,
			entityID:   "",
			domainID:   domainID,
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
			domainID:   domainID,
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
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  1000,
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrLimitSize, http.StatusBadRequest),
		},
		{
			desc:       "retrieve journal with invalid page metadata",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]any{
					"key": make(chan int),
				},
			},
			svcReq:   journal.Page{},
			svcRes:   journal.JournalsPage{},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc:       "retrieve journal with response that cannot be unmarshalled",
			token:      validToken,
			entityType: validEntityType,
			entityID:   validID,
			domainID:   domainID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: journal.Page{
				Offset:     0,
				Limit:      10,
				EntityID:   validID,
				EntityType: journal.GroupEntity,
				Direction:  "desc",
			},
			svcRes: journal.JournalsPage{
				Total: 1,
				Journals: []journal.Journal{{
					ID:         validID,
					Operation:  "create",
					OccurredAt: time.Now(),
					Attributes: validMetadata,
					Metadata: map[string]any{
						"key": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.JournalsPage{},
			err:      errors.NewSDKError(fmt.Errorf("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: fmt.Sprintf("%s_%s", domainID, validID), UserID: validID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RetrieveAll", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Journal(context.Background(), tc.entityType, tc.entityID, tc.domainID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
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

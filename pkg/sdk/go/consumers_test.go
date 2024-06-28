// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala/consumers/notifiers"
	httpapi "github.com/absmach/magistrala/consumers/notifiers/api"
	notmocks "github.com/absmach/magistrala/consumers/notifiers/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ownerID   = testsutil.GenerateUUID(&testing.T{})
	subID     = testsutil.GenerateUUID(&testing.T{})
	sdkSubReq = sdk.Subscription{
		Topic:   "topic",
		Contact: "contact",
	}
	sdkSubRes = sdk.Subscription{
		Topic:   "topic",
		Contact: "contact",
		OwnerID: ownerID,
		ID:      subID,
	}
	notSubReq = notifiers.Subscription{
		Contact: "contact",
		Topic:   "topic",
	}
	notSubRes = notifiers.Subscription{
		Contact: "contact",
		Topic:   "topic",
		OwnerID: ownerID,
		ID:      subID,
	}
)

func setupSubscriptions() (*httptest.Server, *notmocks.Service) {
	nsvc := new(notmocks.Service)
	logger := mglog.NewMock()
	mux := httpapi.MakeHandler(nsvc, logger, instanceID)

	return httptest.NewServer(mux), nsvc
}

func TestCreateSubscription(t *testing.T) {
	ts, nsvc := setupSubscriptions()
	defer ts.Close()

	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc         string
		subscription sdk.Subscription
		token        string
		empty        bool
		id           string
		svcReq       notifiers.Subscription
		svcErr       error
		svcRes       string
		err          errors.SDKError
	}{
		{
			desc:         "create new subscription",
			subscription: sdkSubReq,
			token:        validToken,
			empty:        false,
			svcReq:       notSubReq,
			svcRes:       subID,
			svcErr:       nil,
			err:          nil,
		},
		{
			desc:         "create new subscription with empty token",
			subscription: sdkSubReq,
			token:        "",
			empty:        true,
			svcReq:       notifiers.Subscription{},
			svcRes:       "",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:         "create new subscription with invalid token",
			subscription: sdkSubReq,
			token:        invalidToken,
			empty:        true,
			svcReq:       notSubReq,
			svcRes:       "",
			svcErr:       svcerr.ErrAuthentication,
			err:          errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc: "create new subscription with empty topic",
			subscription: sdk.Subscription{
				Topic:   "",
				Contact: "contact",
			},
			token:  validToken,
			empty:  true,
			svcReq: notifiers.Subscription{},
			svcErr: nil,
			svcRes: "",
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidTopic), http.StatusBadRequest),
		},
		{
			desc: "create new subscription with empty contact",
			subscription: sdk.Subscription{
				Topic:   "topic",
				Contact: "",
			},
			token:  validToken,
			empty:  true,
			svcReq: notifiers.Subscription{},
			svcErr: nil,
			svcRes: "",
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidContact), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := nsvc.On("CreateSubscription", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			loc, err := mgsdk.CreateSubscription(tc.subscription.Topic, tc.subscription.Contact, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.empty, loc == "")
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateSubscription", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewSubscription(t *testing.T) {
	ts, nsvc := setupSubscriptions()
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		subID    string
		token    string
		svcRes   notifiers.Subscription
		svcErr   error
		response sdk.Subscription
		err      errors.SDKError
	}{
		{
			desc:     "view existing subscription",
			subID:    subID,
			token:    validToken,
			svcRes:   notSubRes,
			svcErr:   nil,
			response: sdkSubRes,
			err:      nil,
		},
		{
			desc:     "view non-existent subscription",
			subID:    wrongID,
			token:    validToken,
			svcRes:   notifiers.Subscription{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Subscription{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "view subscription with invalid token",
			subID:    subID,
			token:    invalidToken,
			svcRes:   notifiers.Subscription{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.Subscription{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view subscription with empty token",
			subID:    subID,
			token:    "",
			svcRes:   notifiers.Subscription{},
			svcErr:   nil,
			response: sdk.Subscription{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := nsvc.On("ViewSubscription", mock.Anything, tc.token, tc.subID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ViewSubscription(tc.subID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewSubscription", mock.Anything, tc.token, tc.subID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListSubscription(t *testing.T) {
	ts, nsvc := setupSubscriptions()
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)
	nSubs := 10
	noSubs := []notifiers.Subscription{}
	sdSubs := []sdk.Subscription{}
	for i := 0; i < nSubs; i++ {
		nosub := notifiers.Subscription{
			OwnerID: ownerID,
			Topic:   fmt.Sprintf("topic_%d", i),
			Contact: fmt.Sprintf("contact_%d", i),
		}
		noSubs = append(noSubs, nosub)
		sdsub := sdk.Subscription{
			OwnerID: ownerID,
			Topic:   fmt.Sprintf("topic_%d", i),
			Contact: fmt.Sprintf("contact_%d", i),
		}
		sdSubs = append(sdSubs, sdsub)
	}

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   notifiers.PageMetadata
		svcRes   notifiers.Page
		svcErr   error
		response sdk.SubscriptionPage
		err      errors.SDKError
	}{
		{
			desc:  "list all subscription",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: notifiers.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes: notifiers.Page{
				Total:         10,
				Subscriptions: noSubs,
			},
			svcErr: nil,
			response: sdk.SubscriptionPage{
				PageRes: sdk.PageRes{
					Total: 10,
				},
				Subscriptions: sdSubs,
			},
			err: nil,
		},
		{
			desc:  "list subscription with specific topic",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Topic:  "topic_1",
			},
			svcReq: notifiers.PageMetadata{
				Offset: 0,
				Limit:  10,
				Topic:  "topic_1",
			},
			svcRes: notifiers.Page{
				Total:         uint(len(noSubs[1:2])),
				Subscriptions: noSubs[1:2],
			},
			svcErr: nil,
			response: sdk.SubscriptionPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(sdSubs[1:2])),
				},
				Subscriptions: sdSubs[1:2],
			},
			err: nil,
		},
		{
			desc:  "list subscription with specific contact",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:  0,
				Limit:   10,
				Contact: "contact_1",
			},
			svcReq: notifiers.PageMetadata{
				Offset:  0,
				Limit:   10,
				Contact: "contact_1",
			},
			svcRes: notifiers.Page{
				Total:         uint(len(noSubs[1:2])),
				Subscriptions: noSubs[1:2],
			},
			svcErr: nil,
			response: sdk.SubscriptionPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(sdSubs[1:2])),
				},
				Subscriptions: sdSubs[1:2],
			},
			err: nil,
		},
		{
			desc:  "list subscription with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: notifiers.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcRes:   notifiers.Page{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.SubscriptionPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list subscription with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   notifiers.PageMetadata{},
			svcRes:   notifiers.Page{},
			svcErr:   nil,
			response: sdk.SubscriptionPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list subscription with invalid page metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: sdk.Metadata{
					"key": make(chan int),
				},
			},
			svcReq:   notifiers.PageMetadata{},
			svcRes:   notifiers.Page{},
			svcErr:   nil,
			response: sdk.SubscriptionPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := nsvc.On("ListSubscriptions", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListSubscriptions(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListSubscriptions", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDeleteSubscription(t *testing.T) {
	ts, nsvc := setupSubscriptions()
	defer ts.Close()
	sdkConf := sdk.Config{
		UsersURL:        ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc   string
		subID  string
		token  string
		svcErr error
		err    errors.SDKError
	}{
		{
			desc:   "delete existing subscription",
			subID:  subID,
			token:  validToken,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "delete non-existent subscription",
			subID:  wrongID,
			token:  validToken,
			svcErr: svcerr.ErrRemoveEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:   "delete subscription with invalid token",
			subID:  subID,
			token:  invalidToken,
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "delete subscription with empty token",
			subID:  subID,
			token:  "",
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:   "delete subscription with empty subID",
			subID:  "",
			token:  validToken,
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := nsvc.On("RemoveSubscription", mock.Anything, tc.token, tc.subID).Return(tc.svcErr)
			err := mgsdk.DeleteSubscription(tc.subID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RemoveSubscription", mock.Anything, tc.token, tc.subID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

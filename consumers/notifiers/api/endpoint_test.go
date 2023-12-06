// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/consumers/notifiers"
	httpapi "github.com/absmach/magistrala/consumers/notifiers/api"
	"github.com/absmach/magistrala/consumers/notifiers/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	contact1    = "email1@example.com"
	contact2    = "email2@example.com"
	token       = "token"
	topic       = "topic"
	instanceID  = "5de9b29a-feb9-11ed-be56-0242ac120002"
	validID     = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	notFoundRes   = toJSON(apiutil.ErrorRes{Msg: svcerr.ErrNotFound.Error()})
	unauthRes     = toJSON(apiutil.ErrorRes{Msg: svcerr.ErrAuthentication.Error()})
	invalidRes    = toJSON(apiutil.ErrorRes{Err: apiutil.ErrInvalidQueryParams.Error(), Msg: apiutil.ErrValidation.Error()})
	missingTokRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error(), Msg: apiutil.ErrValidation.Error()})
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newService() (notifiers.Service, *authmocks.Service) {
	auth := new(authmocks.Service)
	repo := mocks.NewRepo(make(map[string]notifiers.Subscription))
	idp := uuid.NewMock()
	notif := mocks.NewNotifier()
	from := "exampleFrom"
	return notifiers.New(auth, repo, idp, notif, from), auth
}

func newServer(svc notifiers.Service) *httptest.Server {
	logger := mglog.NewMock()
	mux := httpapi.MakeHandler(svc, logger, instanceID)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestCreate(t *testing.T) {
	svc, auth := newService()
	ss := newServer(svc)
	defer ss.Close()

	sub := notifiers.Subscription{
		Topic:   topic,
		Contact: contact1,
	}

	data := toJSON(sub)

	emptyTopic := toJSON(notifiers.Subscription{Contact: contact1})
	emptyContact := toJSON(notifiers.Subscription{Topic: "topic123"})

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "add successfully",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    fmt.Sprintf("/subscriptions/%s%012d", uuid.Prefix, 1),
		},
		{
			desc:        "add an existing subscription",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
			location:    "",
		},
		{
			desc:        "add with empty topic",
			req:         emptyTopic,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add with empty contact",
			req:         emptyContact,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add with invalid auth token",
			req:         data,
			contentType: contentType,
			auth:        authmocks.InvalidValue,
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "add with empty auth token",
			req:         data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "add with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)

		req := testRequest{
			client:      ss.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/subscriptions", ss.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))

		repoCall.Unset()
	}
}

func TestView(t *testing.T) {
	svc, auth := newService()
	ss := newServer(svc)
	defer ss.Close()

	sub := notifiers.Subscription{
		Topic:   topic,
		Contact: contact1,
	}
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	id, err := svc.CreateSubscription(context.Background(), token, sub)
	assert.Nil(t, err, fmt.Sprintf("got an error creating id: %s", err))
	repoCall.Unset()

	sr := subRes{
		ID:      id,
		OwnerID: validID,
		Contact: sub.Contact,
		Topic:   sub.Topic,
	}
	data := toJSON(sr)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view successfully",
			id:     id,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view not existing",
			id:     "not existing",
			auth:   token,
			status: http.StatusNotFound,
			res:    notFoundRes,
		},
		{
			desc:   "view with invalid auth token",
			id:     id,
			auth:   authmocks.InvalidValue,
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view with empty auth token",
			id:     id,
			auth:   "",
			status: http.StatusUnauthorized,
			res:    missingTokRes,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)

		req := testRequest{
			client: ss.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/subscriptions/%s", ss.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected request error %s", tc.desc, err))
		body, err := io.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected read error %s", tc.desc, err))
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))

		repoCall.Unset()
	}
}

func TestList(t *testing.T) {
	svc, auth := newService()
	ss := newServer(svc)
	defer ss.Close()

	const numSubs = 100
	var subs []subRes

	for i := 0; i < numSubs; i++ {
		sub := notifiers.Subscription{
			Topic:   fmt.Sprintf("topic.subtopic.%d", i),
			Contact: contact1,
		}
		if i%2 == 0 {
			sub.Contact = contact2
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		id, err := svc.CreateSubscription(context.Background(), token, sub)
		sr := subRes{
			ID:      id,
			OwnerID: validID,
			Contact: sub.Contact,
			Topic:   sub.Topic,
		}
		assert.Nil(t, err, fmt.Sprintf("got an error creating id: %s", err))
		repoCall.Unset()
		subs = append(subs, sr)
	}
	noLimit := toJSON(page{Offset: 5, Limit: 20, Total: numSubs, Subscriptions: subs[5:25]})
	one := toJSON(page{Offset: 0, Limit: 20, Total: 1, Subscriptions: subs[10:11]})

	var contact2Subs []subRes
	for i := 20; i < 40; i += 2 {
		contact2Subs = append(contact2Subs, subs[i])
	}
	contactList := toJSON(page{Offset: 10, Limit: 10, Total: 50, Subscriptions: contact2Subs})

	cases := []struct {
		desc   string
		query  map[string]string
		auth   string
		status int
		res    string
	}{
		{
			desc: "list default limit",
			query: map[string]string{
				"offset": "5",
			},
			auth:   token,
			status: http.StatusOK,
			res:    noLimit,
		},
		{
			desc: "list not existing",
			query: map[string]string{
				"topic": "not-found-topic",
			},
			auth:   token,
			status: http.StatusNotFound,
			res:    notFoundRes,
		},
		{
			desc: "list one with topic",
			query: map[string]string{
				"topic": "topic.subtopic.10",
			},
			auth:   token,
			status: http.StatusOK,
			res:    one,
		},
		{
			desc: "list with contact",
			query: map[string]string{
				"contact": contact2,
				"offset":  "10",
				"limit":   "10",
			},
			auth:   token,
			status: http.StatusOK,
			res:    contactList,
		},
		{
			desc: "list with invalid query",
			query: map[string]string{
				"offset": "two",
			},
			auth:   token,
			status: http.StatusBadRequest,
			res:    invalidRes,
		},
		{
			desc:   "list with invalid auth token",
			auth:   authmocks.InvalidValue,
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "list with empty auth token",
			auth:   "",
			status: http.StatusUnauthorized,
			res:    missingTokRes,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)

		req := testRequest{
			client: ss.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/subscriptions%s", ss.URL, makeQuery(tc.query)),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := io.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: got unexpected body\n", tc.desc))

		repoCall.Unset()
	}
}

func TestRemove(t *testing.T) {
	svc, auth := newService()
	ss := newServer(svc)
	defer ss.Close()

	sub := notifiers.Subscription{
		Topic:   "topic",
		Contact: contact1,
	}
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	id, err := svc.CreateSubscription(context.Background(), token, sub)
	assert.Nil(t, err, fmt.Sprintf("got an error creating id: %s", err))
	repoCall.Unset()

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "remove successfully",
			id:     id,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove not existing",
			id:     "not existing",
			auth:   token,
			status: http.StatusNotFound,
		},
		{
			desc:   "remove empty id",
			id:     "",
			auth:   token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "view with invalid auth token",
			id:     id,
			auth:   authmocks.InvalidValue,
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view with empty auth token",
			id:     id,
			auth:   "",
			status: http.StatusUnauthorized,
			res:    missingTokRes,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: validID}, nil)

		req := testRequest{
			client: ss.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/subscriptions/%s", ss.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

		repoCall.Unset()
	}
}

func makeQuery(m map[string]string) string {
	var ret string
	for k, v := range m {
		ret += fmt.Sprintf("&%s=%s", k, v)
	}
	if ret != "" {
		return fmt.Sprintf("?%s", ret[1:])
	}
	return ""
}

type subRes struct {
	ID      string `json:"id"`
	OwnerID string `json:"owner_id"`
	Contact string `json:"contact"`
	Topic   string `json:"topic"`
}
type page struct {
	Offset        uint     `json:"offset"`
	Limit         int      `json:"limit"`
	Total         uint     `json:"total,omitempty"`
	Subscriptions []subRes `json:"subscriptions,omitempty"`
}

//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/http"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	token       = "token"
	wrongValue  = "wrong_value"
	wrongID     = 0
)

var (
	thing   = things.Thing{Type: "app", Name: "test_app", Metadata: "test_metadata"}
	channel = things.Channel{Name: "test"}
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
		req.Header.Set("Authorization", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newService(tokens map[string]string) things.Service {
	users := mocks.NewUsersService(tokens)
	thingsRepo := mocks.NewThingRepository()
	channelsRepo := mocks.NewChannelRepository(thingsRepo)
	idp := mocks.NewIdentityProvider()
	return things.New(users, thingsRepo, channelsRepo, idp)
}

func newServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestAddThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(thing)
	invalidData := toJSON(things.Thing{Type: "foo"})

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "add valid thing",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/things/1",
		},
		{
			desc:        "add thing with invalid data",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing with invalid auth token",
			req:         data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
			location:    "",
		},
		{
			desc:        "add thing with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(thing)
	invalidData := toJSON(things.Thing{Type: "foo"})
	sth, _ := svc.AddThing(token, thing)

	cases := []struct {
		desc        string
		req         string
		id          uint64
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing thing",
			req:         data,
			id:          sth.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          wrongID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update thing with invalid data",
			req:         invalidData,
			id:          sth.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          sth.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          sth.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          sth.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         "",
			id:          sth.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          sth.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%d", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sth, _ := svc.AddThing(token, thing)
	data := toJSON(sth)

	cases := []struct {
		desc   string
		id     uint64
		auth   string
		status int
		res    string
	}{
		{desc: "view existing thing", id: sth.ID, auth: token, status: http.StatusOK, res: data},
		{desc: "view non-existent thing", id: wrongID, auth: token, status: http.StatusNotFound, res: ""},
		{desc: "view thing by passing invalid token", id: sth.ID, auth: wrongValue, status: http.StatusForbidden, res: ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%d", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := []things.Thing{}
	for i := 0; i < 101; i++ {
		sth, _ := svc.AddThing(token, thing)
		// must be "nulled" due to the JSON serialization that ignores owner
		sth.Owner = ""
		data = append(data, sth)
	}
	thingURL := fmt.Sprintf("%s/things", ts.URL)
	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []things.Thing
	}{
		{
			desc:   "get a list of things",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with invalid token",
			auth:   wrongValue,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", thingURL, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", thingURL, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", thingURL, ""),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things with invalid URL",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?%%"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=5&limit=e"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data map[string][]things.Thing
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data["things"], fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data["things"]))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sth, _ := svc.AddThing(token, thing)

	cases := []struct {
		desc   string
		id     uint64
		auth   string
		status int
	}{
		{desc: "delete existing thing", id: sth.ID, auth: token, status: http.StatusNoContent},
		{desc: "delete non-existent thing", id: wrongID, auth: token, status: http.StatusNoContent},
		{desc: "delete thing with invalid token", id: sth.ID, auth: wrongValue, status: http.StatusForbidden},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%d", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(channel)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "create new channel",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/channels/1",
		},
		{
			desc:        "create new channel with invalid token",
			req:         data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
			location:    "",
		},
		{
			desc:        "create new channel with invalid data format",
			req:         "{",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "create new channel with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/channels/2",
		},
		{
			desc:        "create new channel with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "create new channel without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	updateData := toJSON(map[string]string{"name": "updated_channel"})
	sch, _ := svc.CreateChannel(token, channel)

	cases := []struct {
		desc        string
		req         string
		id          uint64
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing channel",
			req:         updateData,
			id:          sch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existing channel",
			req:         updateData,
			id:          wrongID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update channel with invalid token",
			req:         updateData,
			id:          sch.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update channel with invalid data format",
			req:         "}",
			id:          sch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with empty JSON object",
			req:         "{}",
			id:          sch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update channel with empty request",
			req:         "",
			id:          sch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with missing content type",
			req:         updateData,
			id:          sch.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/channels/%d", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sch, _ := svc.CreateChannel(token, channel)
	data := toJSON(sch)

	cases := []struct {
		desc   string
		id     uint64
		auth   string
		status int
		res    string
	}{
		{desc: "view existing channel", id: sch.ID, auth: token, status: http.StatusOK, res: data},
		{desc: "view non-existent channel", id: wrongID, auth: token, status: http.StatusNotFound, res: ""},
		{desc: "view channel with invalid token", id: sch.ID, auth: wrongValue, status: http.StatusForbidden, res: ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%d", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body := strings.Trim(string(data), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, body))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	channels := []things.Channel{}
	for i := 0; i < 101; i++ {
		sch, _ := svc.CreateChannel(token, channel)
		// must be "nulled" due to the JSON serialization that ignores owner
		sch.Owner = ""
		channels = append(channels, sch)
	}
	channelURL := fmt.Sprintf("%s/channels", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []things.Channel
	}{
		{
			desc:   "get a list of channels",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels with invalid token",
			auth:   wrongValue,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of channels with no offset provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", channelURL, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with no limit provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", channelURL, 1),
			res:    channels[1:11],
		},
		{
			desc:   "get a list of channels with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", channelURL, 0, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of channels with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", channelURL, ""),
			res:    channels[0:10],
		},
		{
			desc:   "get a list of channels with invalid URL",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?%%"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=5&limit=e"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body map[string][]things.Channel
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body["channels"], fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body["channels"]))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sch, _ := svc.CreateChannel(token, channel)

	cases := []struct {
		desc   string
		id     uint64
		auth   string
		status int
	}{
		{desc: "remove channel with invalid token", id: sch.ID, auth: wrongValue, status: http.StatusForbidden},
		{desc: "remove existing channel", id: sch.ID, auth: token, status: http.StatusNoContent},
		{desc: "remove removed channel", id: sch.ID, auth: token, status: http.StatusNoContent},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%d", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestConnect(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ath, _ := svc.AddThing(token, thing)
	ach, _ := svc.CreateChannel(token, channel)
	bch, _ := svc.CreateChannel(otherToken, channel)

	cases := []struct {
		desc    string
		chanID  uint64
		thingID uint64
		auth    string
		status  int
	}{
		{
			desc:    "connect existing thing to existing channel",
			chanID:  ach.ID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusOK,
		},
		{
			desc:    "connect existing thing to non-existent channel",
			chanID:  wrongID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect non-existing thing to existing channel",
			chanID:  ach.ID,
			thingID: wrongID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect existing thing to existing channel with invalid token",
			chanID:  ach.ID,
			thingID: ath.ID,
			auth:    wrongValue,
			status:  http.StatusForbidden,
		},
		{
			desc:    "connect thing from owner to channel of other user",
			chanID:  bch.ID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/channels/%d/things/%d", ts.URL, tc.chanID, tc.thingID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDisconnnect(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ath, _ := svc.AddThing(token, thing)
	ach, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, ach.ID, ath.ID)
	bch, _ := svc.CreateChannel(otherToken, channel)

	cases := []struct {
		desc    string
		chanID  uint64
		thingID uint64
		auth    string
		status  int
	}{
		{
			desc:    "disconnect connected thing from channel",
			chanID:  ach.ID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNoContent,
		},
		{
			desc:    "disconnect non-connected thing from channel",
			chanID:  ach.ID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect non-existent thing from channel",
			chanID:  ach.ID,
			thingID: wrongID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing from non-existent channel",
			chanID:  wrongID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing from channel with invalid token",
			chanID:  ach.ID,
			thingID: ath.ID,
			auth:    wrongValue,
			status:  http.StatusForbidden,
		},
		{
			desc:    "disconnect owner's thing from someone elses channel",
			chanID:  bch.ID,
			thingID: ath.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%d/things/%d", ts.URL, tc.chanID, tc.thingID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

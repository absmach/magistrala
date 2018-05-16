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
	invalid     = "invalid_value"
	wrongID     = "123e4567-e89b-12d3-a456-000000000042"
)

var (
	thing   = things.Thing{Type: "app", Name: "test_app", Payload: "test_payload"}
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
	id := "123e4567-e89b-12d3-a456-000000000001"
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(thing)
	invalidData := toJSON(things.Thing{
		Type:    "foo",
		Name:    "invalid_thing",
		Payload: "some_payload",
	})

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{"add valid thing", data, contentType, token, http.StatusCreated, fmt.Sprintf("/things/%s", id)},
		{"add thing with invalid data", invalidData, contentType, token, http.StatusBadRequest, ""},
		{"add thing with invalid auth token", data, contentType, invalid, http.StatusForbidden, ""},
		{"add thing with invalid request format", "}", contentType, token, http.StatusBadRequest, ""},
		{"add thing with empty JSON request", "{}", contentType, token, http.StatusBadRequest, ""},
		{"add thing with empty request", "", contentType, token, http.StatusBadRequest, ""},
		{"add thing with missing content type", data, "", token, http.StatusUnsupportedMediaType, ""},
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
	invalidData := toJSON(things.Thing{
		Type:    "foo",
		Name:    thing.Name,
		Payload: thing.Payload,
	})
	sth, _ := svc.AddThing(token, thing)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{"update existing thing", data, sth.ID, contentType, token, http.StatusOK},
		{"update non-existent thing", data, wrongID, contentType, token, http.StatusNotFound},
		{"update thing with invalid id", data, invalid, contentType, token, http.StatusNotFound},
		{"update thing with invalid data", invalidData, sth.ID, contentType, token, http.StatusBadRequest},
		{"update thing with invalid user token", data, sth.ID, contentType, invalid, http.StatusForbidden},
		{"update thing with invalid data format", "{", sth.ID, contentType, token, http.StatusBadRequest},
		{"update thing with empty JSON request", "{}", sth.ID, contentType, token, http.StatusBadRequest},
		{"update thing with empty request", "", sth.ID, contentType, token, http.StatusBadRequest},
		{"update thing with missing content type", data, sth.ID, "", token, http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
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
		id     string
		auth   string
		status int
		res    string
	}{
		{"view existing thing", sth.ID, token, http.StatusOK, data},
		{"view non-existent thing", wrongID, token, http.StatusNotFound, ""},
		{"view thing by passing invalid id", invalid, token, http.StatusNotFound, ""},
		{"view thing by passing invalid token", sth.ID, invalid, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
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
		{"get a list of things", token, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 5), data[0:5]},
		{"get a list of things with invalid token", invalid, http.StatusForbidden, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1), nil},
		{"get a list of things with invalid offset", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, -1, 5), nil},
		{"get a list of things with invalid limit", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, -5), nil},
		{"get a list of things with zero limit", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, 0), nil},
		{"get a list of things with no offset provided", token, http.StatusOK, fmt.Sprintf("%s?limit=%d", thingURL, 5), data[0:5]},
		{"get a list of things with no limit provided", token, http.StatusOK, fmt.Sprintf("%s?offset=%d", thingURL, 1), data[1:11]},
		{"get a list of things with redundant query params", token, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d&value=something", thingURL, 0, 5), data[0:5]},
		{"get a list of things with limit greater than max", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 110), nil},
		{"get a list of things with default URL", token, http.StatusOK, fmt.Sprintf("%s%s", thingURL, ""), data[0:10]},
		{"get a list of things with invalid URL", token, http.StatusBadRequest, fmt.Sprintf("%s%s", thingURL, "?%%"), nil},
		{"get a list of things with invalid number of params", token, http.StatusBadRequest, fmt.Sprintf("%s%s", thingURL, "?offset=4&limit=4&limit=5&offset=5"), nil},
		{"get a list of things with invalid offset", token, http.StatusBadRequest, fmt.Sprintf("%s%s", thingURL, "?offset=e&limit=5"), nil},
		{"get a list of things with invalid limit", token, http.StatusBadRequest, fmt.Sprintf("%s%s", thingURL, "?offset=5&limit=e"), nil},
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
		assert.ElementsMatch(t, tc.res, data["things"], fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data["things"]))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sth, _ := svc.AddThing(token, thing)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{"delete existing thing", sth.ID, token, http.StatusNoContent},
		{"delete non-existent thing", wrongID, token, http.StatusNoContent},
		{"delete thing with invalid id", invalid, token, http.StatusNoContent},
		{"delete thing with invalid token", sth.ID, invalid, http.StatusForbidden},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateChannel(t *testing.T) {
	id := "123e4567-e89b-12d3-a456-000000000001"
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
		{"create new channel", data, contentType, token, http.StatusCreated, fmt.Sprintf("/channels/%s", id)},
		{"create new channel with invalid token", data, contentType, invalid, http.StatusForbidden, ""},
		{"create new channel with invalid data format", "{", contentType, token, http.StatusBadRequest, ""},
		{"create new channel with empty JSON request", "{}", contentType, token, http.StatusCreated, "/channels/123e4567-e89b-12d3-a456-000000000002"},
		{"create new channel with empty request", "", contentType, token, http.StatusBadRequest, ""},
		{"create new channel with missing content type", data, "", token, http.StatusUnsupportedMediaType, ""},
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

	updateData := toJSON(map[string]string{
		"name": "updated_channel",
	})
	sch, _ := svc.CreateChannel(token, channel)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{"update existing channel", updateData, sch.ID, contentType, token, http.StatusOK},
		{"update non-existing channel", updateData, wrongID, contentType, token, http.StatusNotFound},
		{"update channel with invalid token", updateData, sch.ID, contentType, invalid, http.StatusForbidden},
		{"update channel with invalid id", updateData, invalid, contentType, token, http.StatusNotFound},
		{"update channel with invalid data format", "}", sch.ID, contentType, token, http.StatusBadRequest},
		{"update channel with empty JSON object", "{}", sch.ID, contentType, token, http.StatusOK},
		{"update channel with empty request", "", sch.ID, contentType, token, http.StatusBadRequest},
		{"update channel with missing content type", updateData, sch.ID, "", token, http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
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
		id     string
		auth   string
		status int
		res    string
	}{
		{"view existing channel", sch.ID, token, http.StatusOK, data},
		{"view non-existent channel", wrongID, token, http.StatusNotFound, ""},
		{"view channel with invalid id", invalid, token, http.StatusNotFound, ""},
		{"view channel with invalid token", sch.ID, invalid, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
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
		{"get a list of channels", token, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 6), channels[0:6]},
		{"get a list of channels with invalid token", invalid, http.StatusForbidden, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1), nil},
		{"get a list of channels with invalid offset", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5), nil},
		{"get a list of channels with invalid limit", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5), nil},
		{"get a list of channels with zero limit", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 1, 0), nil},
		{"get a list of channels with no offset provided", token, http.StatusOK, fmt.Sprintf("%s?limit=%d", channelURL, 5), channels[0:5]},
		{"get a list of channels with no limit provided", token, http.StatusOK, fmt.Sprintf("%s?offset=%d", channelURL, 1), channels[1:11]},
		{"get a list of channels with redundant query params", token, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d&value=something", channelURL, 0, 5), channels[0:5]},
		{"get a list of channels with limit greater than max", token, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 110), nil},
		{"get a list of channels with default URL", token, http.StatusOK, fmt.Sprintf("%s%s", channelURL, ""), channels[0:10]},
		{"get a list of channels with invalid URL", token, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?%%"), nil},
		{"get a list of channels with invalid number of params", token, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=4&limit=4&limit=5&offset=5"), nil},
		{"get a list of channels with invalid offset", token, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=e&limit=5"), nil},
		{"get a list of channels with invalid limit", token, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=5&limit=e"), nil},
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
		assert.ElementsMatch(t, tc.res, body["channels"], fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, body["channels"]))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	sch, _ := svc.CreateChannel(token, channel)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{"remove existing channel", sch.ID, token, http.StatusNoContent},
		{"remove non-existent channel", sch.ID, token, http.StatusNoContent},
		{"remove channel with invalid id", wrongID, token, http.StatusNoContent},
		{"remove channel with invalid token", sch.ID, invalid, http.StatusForbidden},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
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
		chanID  string
		thingID string
		auth    string
		status  int
	}{
		{"connect existing thing to existing channel", ach.ID, ath.ID, token, http.StatusOK},
		{"connect existing thing to non-existent channel", wrongID, ath.ID, token, http.StatusNotFound},
		{"connect thing with invalid id to channel", ach.ID, invalid, token, http.StatusNotFound},
		{"connect thing to channel with invalid id", invalid, ath.ID, token, http.StatusNotFound},
		{"connect existing thing to existing channel with invalid token", ach.ID, ath.ID, invalid, http.StatusForbidden},
		{"connect thing from owner to channel of other user", bch.ID, ath.ID, token, http.StatusNotFound},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/channels/%s/things/%s", ts.URL, tc.chanID, tc.thingID),
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
		chanID  string
		thingID string
		auth    string
		status  int
	}{
		{"disconnect connected thing from channel", ach.ID, ath.ID, token, http.StatusNoContent},
		{"disconnect non-connected thing from channel", ach.ID, ath.ID, token, http.StatusNotFound},
		{"disconnect non-existent thing from channel", ach.ID, invalid, token, http.StatusNotFound},
		{"disconnect thing from non-existent channel", invalid, ath.ID, token, http.StatusNotFound},
		{"disconnect thing from channel with invalid token", ach.ID, ath.ID, invalid, http.StatusForbidden},
		{"disconnect owner's thing from someone elses channel", bch.ID, ath.ID, token, http.StatusNotFound},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s/things/%s", ts.URL, tc.chanID, tc.thingID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

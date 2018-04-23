package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/api"
	"github.com/mainflux/mainflux/manager/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	contentType  = "application/json"
	invalidEmail = "userexample.com"
	wrongID      = "123e4567-e89b-12d3-a456-000000000042"
	id           = "123e4567-e89b-12d3-a456-000000000001"
)

var (
	user    = manager.User{"user@example.com", "password"}
	client  = manager.Client{Type: "app", Name: "test_app", Payload: "test_payload"}
	channel = manager.Channel{Name: "test"}
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

func newService() manager.Service {
	users := mocks.NewUserRepository()
	clients := mocks.NewClientRepository()
	channels := mocks.NewChannelRepository(clients)
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()

	return manager.New(users, clients, channels, hasher, idp)
}

func newServer(svc manager.Service) *httptest.Server {
	mux := api.MakeHandler(svc)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestRegister(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(user)
	invalidData := toJSON(manager.User{Email: invalidEmail, Password: "password"})

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
	}{
		{"register new user", data, contentType, http.StatusCreated},
		{"register existing user", data, contentType, http.StatusConflict},
		{"register user with invalid email address", invalidData, contentType, http.StatusBadRequest},
		{"register user with invalid request format", "{", contentType, http.StatusBadRequest},
		{"register user with empty JSON request", "{}", contentType, http.StatusBadRequest},
		{"register user with empty request", "", contentType, http.StatusBadRequest},
		{"register user with missing content type", data, "", http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	tokenData := toJSON(map[string]string{"token": user.Email})
	data := toJSON(user)
	invalidEmailData := toJSON(manager.User{Email: invalidEmail, Password: "password"})
	invalidData := toJSON(manager.User{"user@example.com", "invalid_password"})
	nonexistentData := toJSON(manager.User{"non-existentuser@example.com", "pass"})
	svc.Register(user)

	cases := []struct {
		desc        string
		req         string
		contentType string
		status      int
		res         string
	}{
		{"login with valid credentials", data, contentType, http.StatusCreated, tokenData},
		{"login with invalid credentials", invalidData, contentType, http.StatusForbidden, ""},
		{"login with invalid email address", invalidEmailData, contentType, http.StatusBadRequest, ""},
		{"login non-existent user", nonexistentData, contentType, http.StatusForbidden, ""},
		{"login with invalid request format", "{", contentType, http.StatusBadRequest, ""},
		{"login with empty JSON request", "{}", contentType, http.StatusBadRequest, ""},
		{"login with empty request", "", contentType, http.StatusBadRequest, ""},
		{"login with missing content type", data, "", http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/tokens", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		token := strings.Trim(string(body), "\n")

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, token, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, token))
	}
}

func TestAddClient(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	data := toJSON(client)
	invalidData := toJSON(manager.Client{
		Type:    "foo",
		Name:    "invalid_client",
		Payload: "some_payload",
	})
	svc.Register(user)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{"add valid client", data, contentType, user.Email, http.StatusCreated, fmt.Sprintf("/clients/%s", id)},
		{"add client with invalid data", invalidData, contentType, user.Email, http.StatusBadRequest, ""},
		{"add client with invalid auth token", data, contentType, "invalid_token", http.StatusForbidden, ""},
		{"add client with invalid request format", "}", contentType, user.Email, http.StatusBadRequest, ""},
		{"add client with empty JSON request", "{}", contentType, user.Email, http.StatusBadRequest, ""},
		{"add client with empty request", "", contentType, user.Email, http.StatusBadRequest, ""},
		{"add client with missing content type", data, "", user.Email, http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      cli,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/clients", ts.URL),
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

func TestUpdateClient(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	data := toJSON(client)
	invalidData := toJSON(manager.Client{
		Type:    "foo",
		Name:    client.Name,
		Payload: client.Payload,
	})
	svc.Register(user)
	id, _ := svc.AddClient(user.Email, client)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{"update existing client", data, id, contentType, user.Email, http.StatusOK},
		{"update non-existent client", data, wrongID, contentType, user.Email, http.StatusNotFound},
		{"update client with invalid id", data, "1", contentType, user.Email, http.StatusNotFound},
		{"update client with invalid data", invalidData, id, contentType, user.Email, http.StatusBadRequest},
		{"update client with invalid user token", data, id, contentType, invalidEmail, http.StatusForbidden},
		{"update client with invalid data format", "{", id, contentType, user.Email, http.StatusBadRequest},
		{"update client with empty JSON request", "{}", id, contentType, user.Email, http.StatusBadRequest},
		{"update client with empty request", "", id, contentType, user.Email, http.StatusBadRequest},
		{"update client with missing content type", data, id, "", user.Email, http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      cli,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/clients/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewClient(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	id, _ := svc.AddClient(user.Email, client)

	client.ID = id
	client.Key = id
	data := toJSON(client)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{"view existing client", id, user.Email, http.StatusOK, data},
		{"view non-existent client", wrongID, user.Email, http.StatusNotFound, ""},
		{"view client by passing invalid id", "1", user.Email, http.StatusNotFound, ""},
		{"view client by passing invalid token", id, invalidEmail, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/clients/%s", ts.URL, tc.id),
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

func TestListClients(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	noClientsUser := manager.User{Email: "no_clients_user@example.com", Password: user.Password}
	svc.Register(noClientsUser)
	clients := []manager.Client{}
	for i := 0; i < 101; i++ {
		id, _ := svc.AddClient(user.Email, client)
		client.ID = id
		client.Key = id
		clients = append(clients, client)
	}
	clientURL := fmt.Sprintf("%s/clients", ts.URL)
	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []manager.Client
	}{
		{"get a list of clients", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, 0, 5), clients[0:5]},
		{"get a list of clients with invalid token", invalidEmail, http.StatusForbidden, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, 0, 1), nil},
		{"get a list of clients with invalid offset", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, -1, 5), nil},
		{"get a list of clients with invalid limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, 1, -5), nil},
		{"get a list of clients with zero limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, 1, 0), nil},
		{"get a list of clients with no offset provided", user.Email, http.StatusOK, fmt.Sprintf("%s?limit=%d", clientURL, 5), clients[0:5]},
		{"get a list of clients with no limit provided", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d", clientURL, 1), clients[1:11]},
		{"get a list of clients with redundant query params", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d&value=something", clientURL, 0, 5), clients[0:5]},
		{"get a list of clients with limit greater than max", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", clientURL, 0, 110), nil},
		{"get a list of clients with default URL", user.Email, http.StatusOK, fmt.Sprintf("%s%s", clientURL, ""), clients[0:10]},
		{"get a list of clients with invalid URL", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", clientURL, "?%%"), nil},
		{"get a list of clients with invalid number of params", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", clientURL, "?offset=4&limit=4&limit=5&offset=5"), nil},
		{"get a list of clients with invalid offset", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", clientURL, "?offset=e&limit=5"), nil},
		{"get a list of clients with invalid limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", clientURL, "?offset=5&limit=e"), nil},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data map[string][]manager.Client
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data["clients"], fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data["clients"]))
	}
}

func TestRemoveClient(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	id, _ := svc.AddClient(user.Email, client)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{"delete existing client", id, user.Email, http.StatusNoContent},
		{"delete non-existent client", wrongID, user.Email, http.StatusNoContent},
		{"delete client with invalid id", "1", user.Email, http.StatusNoContent},
		{"delete client with invalid token", id, invalidEmail, http.StatusForbidden},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/clients/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	data := toJSON(channel)
	svc.Register(user)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{"create new channel", data, contentType, user.Email, http.StatusCreated, fmt.Sprintf("/channels/%s", id)},
		{"create new channel with invalid token", data, contentType, invalidEmail, http.StatusForbidden, ""},
		{"create new channel with invalid data format", "{", contentType, user.Email, http.StatusBadRequest, ""},
		{"create new channel with empty JSON request", "{}", contentType, user.Email, http.StatusCreated, "/channels/123e4567-e89b-12d3-a456-000000000002"},
		{"create new channel with empty request", "", contentType, user.Email, http.StatusBadRequest, ""},
		{"create new channel with missing content type", data, "", user.Email, http.StatusUnsupportedMediaType, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
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
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	updateData := toJSON(map[string]string{
		"name": "updated_channel",
	})
	svc.Register(user)
	id, _ := svc.CreateChannel(user.Email, channel)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{"update existing channel", updateData, id, contentType, user.Email, http.StatusOK},
		{"update non-existing channel", updateData, wrongID, contentType, user.Email, http.StatusNotFound},
		{"update channel with invalid token", updateData, id, contentType, invalidEmail, http.StatusForbidden},
		{"update channel with invalid id", updateData, "1", contentType, user.Email, http.StatusNotFound},
		{"update channel with invalid data format", "}", id, contentType, user.Email, http.StatusBadRequest},
		{"update channel with empty JSON object", "{}", id, contentType, user.Email, http.StatusOK},
		{"update channel with empty request", "", id, contentType, user.Email, http.StatusBadRequest},
		{"update channel with missing content type", updateData, id, "", user.Email, http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
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
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	svc.Register(user)
	id, _ := svc.CreateChannel(user.Email, channel)
	channel.ID = id
	data := toJSON(channel)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{"view existing channel", id, user.Email, http.StatusOK, data},
		{"view non-existent channel", wrongID, user.Email, http.StatusNotFound, ""},
		{"view channel with invalid id", "1", user.Email, http.StatusNotFound, ""},
		{"view channel with invalid token", id, invalidEmail, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
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
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	svc.Register(user)
	channels := []manager.Channel{}
	for i := 0; i < 101; i++ {
		id, _ := svc.CreateChannel(user.Email, channel)
		channel.ID = id
		channels = append(channels, channel)
	}
	channelURL := fmt.Sprintf("%s/channels", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []manager.Channel
	}{
		{"get a list of channels", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 6), channels[0:6]},
		{"get a list of channels with invalid token", invalidEmail, http.StatusForbidden, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1), nil},
		{"get a list of channels with invalid offset", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5), nil},
		{"get a list of channels with invalid limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5), nil},
		{"get a list of channels with zero limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 1, 0), nil},
		{"get a list of channels with no offset provided", user.Email, http.StatusOK, fmt.Sprintf("%s?limit=%d", channelURL, 5), channels[0:5]},
		{"get a list of channels with no limit provided", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d", channelURL, 1), channels[1:11]},
		{"get a list of channels with redundant query params", user.Email, http.StatusOK, fmt.Sprintf("%s?offset=%d&limit=%d&value=something", channelURL, 0, 5), channels[0:5]},
		{"get a list of channels with limit greater than max", user.Email, http.StatusBadRequest, fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 110), nil},
		{"get a list of channels with default URL", user.Email, http.StatusOK, fmt.Sprintf("%s%s", channelURL, ""), channels[0:10]},
		{"get a list of channels with invalid URL", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?%%"), nil},
		{"get a list of channels with invalid number of params", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=4&limit=4&limit=5&offset=5"), nil},
		{"get a list of channels with invalid offset", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=e&limit=5"), nil},
		{"get a list of channels with invalid limit", user.Email, http.StatusBadRequest, fmt.Sprintf("%s%s", channelURL, "?offset=5&limit=e"), nil},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body map[string][]manager.Channel
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body["channels"], fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, body["channels"]))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	svc.Register(user)
	id, _ := svc.CreateChannel(user.Email, channel)
	channel.ID = id

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{"remove existing channel", channel.ID, user.Email, http.StatusNoContent},
		{"remove non-existent channel", channel.ID, user.Email, http.StatusNoContent},
		{"remove channel with invalid id", wrongID, user.Email, http.StatusNoContent},
		{"remove channel with invalid token", channel.ID, invalidEmail, http.StatusForbidden},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
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
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	clientID, _ := svc.AddClient(user.Email, client)
	chanID, _ := svc.CreateChannel(user.Email, channel)

	otherUser := manager.User{Email: "other_user@example.com", Password: "password"}
	svc.Register(otherUser)
	otherClientID, _ := svc.AddClient(otherUser.Email, client)
	otherChanID, _ := svc.CreateChannel(otherUser.Email, channel)

	cases := []struct {
		desc     string
		chanID   string
		clientID string
		auth     string
		status   int
	}{
		{"connect existing client to existing channel", chanID, clientID, user.Email, http.StatusOK},
		{"connect existing client to non-existent channel", wrongID, clientID, user.Email, http.StatusNotFound},
		{"connect client with invalid id to channel", chanID, "1", user.Email, http.StatusNotFound},
		{"connect client to channel with invalid id", "1", clientID, user.Email, http.StatusNotFound},
		{"connect existing client to existing channel with invalid token", chanID, clientID, invalidEmail, http.StatusForbidden},
		{"connect client from owner to channel of other user", otherChanID, clientID, user.Email, http.StatusNotFound},
		{"connect client from other user to owner's channel", chanID, otherClientID, user.Email, http.StatusNotFound},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/channels/%s/clients/%s", ts.URL, tc.chanID, tc.clientID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDisconnnect(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	clientID, _ := svc.AddClient(user.Email, client)
	chanID, _ := svc.CreateChannel(user.Email, channel)
	svc.Connect(user.Email, chanID, clientID)
	otherUser := manager.User{Email: "other_user@example.com", Password: "password"}
	svc.Register(otherUser)
	otherClientID, _ := svc.AddClient(otherUser.Email, client)
	otherChanID, _ := svc.CreateChannel(otherUser.Email, channel)
	svc.Connect(otherUser.Email, otherChanID, otherClientID)

	cases := []struct {
		desc     string
		chanID   string
		clientID string
		auth     string
		status   int
	}{
		{"disconnect connected client from channel", chanID, clientID, user.Email, http.StatusNoContent},
		{"disconnect non-connected client from channel", chanID, clientID, user.Email, http.StatusNotFound},
		{"disconnect non-existent client from channel", chanID, "1", user.Email, http.StatusNotFound},
		{"disconnect client from non-existent channel", "1", clientID, user.Email, http.StatusNotFound},
		{"disconnect client from channel with invalid token", chanID, clientID, invalidEmail, http.StatusForbidden},
		{"disconnect owner's client from someone elses channel", otherChanID, clientID, user.Email, http.StatusNotFound},
		{"disconnect other's client from owner's channel", chanID, otherClientID, user.Email, http.StatusNotFound},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s/clients/%s", ts.URL, tc.chanID, tc.clientID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestIdentity(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	clientID, _ := svc.AddClient(user.Email, client)

	cases := []struct {
		desc     string
		key      string
		status   int
		clientID string
	}{
		{"get client id using existing client key", clientID, http.StatusOK, clientID},
		{"get client id using non-existent client key", "", http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/access-grant", ts.URL),
			token:  tc.key,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		clientID := res.Header.Get("X-client-id")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.clientID, clientID, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.clientID, clientID))
	}
}

func TestCanAccess(t *testing.T) {
	svc := newService()
	ts := newServer(svc)
	defer ts.Close()
	cli := ts.Client()

	svc.Register(user)
	clientID, _ := svc.AddClient(user.Email, client)
	notConnectedClientID, _ := svc.AddClient(user.Email, client)
	chanID, _ := svc.CreateChannel(user.Email, channel)
	svc.Connect(user.Email, chanID, clientID)

	cases := []struct {
		desc      string
		chanID    string
		clientKey string
		status    int
		clientID  string
	}{
		{"check access to existing channel given connected client", chanID, clientID, http.StatusOK, clientID},
		{"check access to existing channel given not connected client", chanID, notConnectedClientID, http.StatusForbidden, ""},
		{"check access to existing channel given non-existent client", chanID, "invalid_token", http.StatusForbidden, ""},
		{"check access to non-existent channel given existing client", "invalid_token", clientID, http.StatusForbidden, ""},
	}

	for _, tc := range cases {
		req := testRequest{
			client: cli,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%s/access-grant", ts.URL, tc.chanID),
			token:  tc.clientKey,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		clientID := res.Header.Get("X-client-id")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.clientID, clientID, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.clientID, clientID))
	}
}

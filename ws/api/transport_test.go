package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/api"
	"github.com/mainflux/mainflux/ws/mocks"
	broker "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
)

const (
	chanID   = "123e4567-e89b-12d3-a456-000000000001"
	token    = "token"
	protocol = "ws"
)

var (
	msg     = []byte(`[{"n":"current","t":-5,"v":1.2}]`)
	channel = ws.Channel{make(chan mainflux.RawMessage), make(chan bool)}
)

func newService() ws.Service {
	subs := map[string]ws.Channel{chanID: channel}
	pubsub := mocks.NewService(subs, broker.ErrConnectionClosed)
	return ws.New(pubsub)
}

func newHTTPServer(svc ws.Service, cc mainflux.ClientsServiceClient) *httptest.Server {
	mux := api.MakeHandler(svc, cc, log.New(os.Stdout))
	return httptest.NewServer(mux)
}

func newClientsClient() mainflux.ClientsServiceClient {
	clientID := chanID
	return mocks.NewClientsClient(map[string]string{token: clientID})
}

func makeURL(tsURL, chanID, auth string, header bool) string {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol
	if header {
		return fmt.Sprintf("%s/channels/%s/messages", u, chanID)
	}
	return fmt.Sprintf("%s/channels/%s/messages?authorization=%s", u, chanID, auth)
}

func handshake(tsURL, chanID, token string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", token)
	}
	url := makeURL(tsURL, chanID, token, addHeader)
	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	return conn, resp, err
}

func TestHandshake(t *testing.T) {
	clientsClient := newClientsClient()
	svc := newService()
	ts := newHTTPServer(svc, clientsClient)
	defer ts.Close()

	cases := []struct {
		desc   string
		chanID string
		header bool
		token  string
		status int
		msg    []byte
	}{
		{"connect and send message", chanID, true, token, http.StatusSwitchingProtocols, msg},
		{"connect to non-existent channel", "123e4567-e89b-12d3-a456-000000000042", true, token, http.StatusSwitchingProtocols, []byte{}},
		{"connect with invalid token", chanID, true, "", http.StatusForbidden, []byte{}},
		{"connect with invalid channel id", "1", true, token, http.StatusNotFound, []byte{}},
		{"connect and send message with token as query parameter", chanID, false, token, http.StatusSwitchingProtocols, msg},
		{"connect and send message that cannot be published", chanID, true, token, http.StatusSwitchingProtocols, []byte{}},
	}

	for _, tc := range cases {
		conn, res, err := handshake(ts.URL, tc.chanID, tc.token, tc.header)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d\n", tc.desc, tc.status, res.StatusCode))
		if err != nil {
			continue
		}
		err = conn.WriteMessage(websocket.TextMessage, tc.msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
	}
}

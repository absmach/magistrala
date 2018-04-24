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
	manager "github.com/mainflux/mainflux/manager/client"
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

func newHTTPServer(svc ws.Service, mc manager.ManagerClient) *httptest.Server {
	mux := api.MakeHandler(svc, mc, log.New(os.Stdout))
	return httptest.NewServer(mux)
}

func newManagerServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(authorize))
}

func authorize(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "" {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func newManagerClient(url string) manager.ManagerClient {
	return manager.NewClient(url)
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
	mcServer := newManagerServer()
	mc := newManagerClient(mcServer.URL)
	svc := newService()
	ts := newHTTPServer(svc, mc)
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

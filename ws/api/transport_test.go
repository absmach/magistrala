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
	chanID   = 1
	token    = "token"
	protocol = "ws"
)

var (
	msg     = []byte(`[{"n":"current","t":-5,"v":1.2}]`)
	channel = ws.NewChannel()
)

func newService() ws.Service {
	subs := map[uint64]*ws.Channel{chanID: channel}
	pubsub := mocks.NewService(subs, broker.ErrConnectionClosed)
	return ws.New(pubsub)
}

func newHTTPServer(svc ws.Service, tc mainflux.ThingsServiceClient) *httptest.Server {
	mux := api.MakeHandler(svc, tc, log.New(os.Stdout))
	return httptest.NewServer(mux)
}

func newThingsClient() mainflux.ThingsServiceClient {
	thingID := uint64(chanID)
	return mocks.NewThingsClient(map[string]uint64{token: thingID})
}

func makeURL(tsURL string, chanID int64, auth string, header bool) string {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol
	if header {
		return fmt.Sprintf("%s/channels/%d/messages", u, chanID)
	}
	return fmt.Sprintf("%s/channels/%d/messages?authorization=%s", u, chanID, auth)
}

func handshake(tsURL string, chanID int64, token string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", token)
	}
	url := makeURL(tsURL, chanID, token, addHeader)
	return websocket.DefaultDialer.Dial(url, header)
}

func TestHandshake(t *testing.T) {
	thingsClient := newThingsClient()
	svc := newService()
	ts := newHTTPServer(svc, thingsClient)
	defer ts.Close()

	cases := []struct {
		desc   string
		chanID int64
		header bool
		token  string
		status int
		msg    []byte
	}{
		{"connect and send message", chanID, true, token, http.StatusSwitchingProtocols, msg},
		{"connect to non-existent channel", 0, true, token, http.StatusSwitchingProtocols, []byte{}},
		{"connect to invalid channel id", -5, true, token, http.StatusNotFound, []byte{}},
		{"connect with empty token", chanID, true, "", http.StatusForbidden, []byte{}},
		{"connect with invalid token", chanID, true, "invalid", http.StatusForbidden, []byte{}},
		{"connect unable to authorize", chanID, true, mocks.ServiceErrToken, http.StatusServiceUnavailable, []byte{}},
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

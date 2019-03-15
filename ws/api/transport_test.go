//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

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
	id       = "1"
	token    = "token"
	protocol = "ws"
)

var (
	msg     = []byte(`[{"n":"current","t":-5,"v":1.2}]`)
	channel = ws.NewChannel()
)

func newService() ws.Service {
	subs := map[string]*ws.Channel{id: channel}
	pubsub := mocks.NewService(subs, broker.ErrConnectionClosed)
	return ws.New(pubsub)
}

func newHTTPServer(svc ws.Service, tc mainflux.ThingsServiceClient) *httptest.Server {
	logger, _ := log.New(os.Stdout, log.Info.String())
	mux := api.MakeHandler(svc, tc, logger)
	return httptest.NewServer(mux)
}

func newThingsClient() mainflux.ThingsServiceClient {
	return mocks.NewThingsClient(map[string]string{token: id})
}

func makeURL(tsURL, chanID, subtopic, auth string, header bool) string {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol
	subtopicPart := ""
	if subtopic != "" {
		subtopicPart = fmt.Sprintf("/%s", subtopic)
	}
	if header {
		return fmt.Sprintf("%s/channels/%s/messages%s", u, chanID, subtopicPart)
	}
	return fmt.Sprintf("%s/channels/%s/messages%s?authorization=%s", u, chanID, subtopicPart, auth)
}

func handshake(tsURL, chanID, subtopic, token string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", token)
	}
	url := makeURL(tsURL, chanID, subtopic, token, addHeader)
	return websocket.DefaultDialer.Dial(url, header)
}

func TestHandshake(t *testing.T) {
	thingsClient := newThingsClient()
	svc := newService()
	ts := newHTTPServer(svc, thingsClient)
	defer ts.Close()

	cases := []struct {
		desc     string
		chanID   string
		subtopic string
		header   bool
		token    string
		status   int
		msg      []byte
	}{
		{"connect and send message", id, "", true, token, http.StatusSwitchingProtocols, msg},
		{"connect to non-existent channel", "0", "", true, token, http.StatusSwitchingProtocols, []byte{}},
		{"connect to invalid channel id", "", "", true, token, http.StatusBadRequest, []byte{}},
		{"connect with empty token", id, "", true, "", http.StatusForbidden, []byte{}},
		{"connect with invalid token", id, "", true, "invalid", http.StatusForbidden, []byte{}},
		{"connect unable to authorize", id, "", true, mocks.ServiceErrToken, http.StatusServiceUnavailable, []byte{}},
		{"connect and send message with token as query parameter", id, "", false, token, http.StatusSwitchingProtocols, msg},
		{"connect and send message that cannot be published", id, "", true, token, http.StatusSwitchingProtocols, []byte{}},
		{"connect and send message to subtopic", id, "subtopic", true, token, http.StatusSwitchingProtocols, msg},
		{"connect and send message to subtopic with invalid name", id, "sub//topic", true, token, http.StatusBadRequest, msg},
		{"connect and send message to nested subtopic", id, "subtopic/nested", true, token, http.StatusSwitchingProtocols, msg},
		{"connect and send message to all subtopics", id, ">", true, token, http.StatusSwitchingProtocols, msg},
	}

	for _, tc := range cases {
		conn, res, err := handshake(ts.URL, tc.chanID, tc.subtopic, tc.token, tc.header)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d\n", tc.desc, tc.status, res.StatusCode))
		if err != nil {
			continue
		}
		err = conn.WriteMessage(websocket.TextMessage, tc.msg)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
	}
}

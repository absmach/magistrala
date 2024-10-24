// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	chmocks "github.com/absmach/magistrala/channels/mocks"
	climocks "github.com/absmach/magistrala/clients/mocks"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	mglog "github.com/absmach/magistrala/logger"
	authnMocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/messaging/mocks"
	"github.com/absmach/magistrala/ws"
	"github.com/absmach/magistrala/ws/api"
	"github.com/absmach/mgate/pkg/session"
	"github.com/absmach/mgate/pkg/websockets"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	chanID     = "30315311-56ba-484d-b500-c1e08305511f"
	id         = "1"
	clientKey  = "c02ff576-ccd5-40f6-ba5f-c85377aad529"
	protocol   = "ws"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
)

var msg = []byte(`[{"n":"current","t":-1,"v":1.6}]`)

func newService(clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient) (ws.Service, *mocks.PubSub) {
	pubsub := new(mocks.PubSub)
	return ws.New(clients, channels, pubsub), pubsub
}

func newHTTPServer(svc ws.Service) *httptest.Server {
	mux := api.MakeHandler(context.Background(), svc, mglog.NewMock(), instanceID)
	return httptest.NewServer(mux)
}

func newProxyHTPPServer(svc session.Handler, targetServer *httptest.Server) (*httptest.Server, error) {
	turl := strings.ReplaceAll(targetServer.URL, "http", "ws")
	mp, err := websockets.NewProxy("", turl, mglog.NewMock(), svc)
	if err != nil {
		return nil, err
	}
	return httptest.NewServer(http.HandlerFunc(mp.Handler)), nil
}

func makeURL(tsURL, chanID, subtopic, clientKey string, header bool) (string, error) {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol

	if chanID == "0" || chanID == "" {
		if header {
			return fmt.Sprintf("%s/channels/%s/messages", u, chanID), fmt.Errorf("invalid channel id")
		}
		return fmt.Sprintf("%s/channels/%s/messages?authorization=%s", u, chanID, clientKey), fmt.Errorf("invalid channel id")
	}

	subtopicPart := ""
	if subtopic != "" {
		subtopicPart = fmt.Sprintf("/%s", subtopic)
	}
	if header {
		return fmt.Sprintf("%s/channels/%s/messages%s", u, chanID, subtopicPart), nil
	}

	return fmt.Sprintf("%s/channels/%s/messages%s?authorization=%s", u, chanID, subtopicPart, clientKey), nil
}

func handshake(tsURL, chanID, subtopic, clientKey string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", clientKey)
	}

	turl, _ := makeURL(tsURL, chanID, subtopic, clientKey, addHeader)
	conn, res, errRet := websocket.DefaultDialer.Dial(turl, header)

	return conn, res, errRet
}

func TestHandshake(t *testing.T) {
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)
	authn := new(authnMocks.Authentication)
	svc, pubsub := newService(clients, channels)
	target := newHTTPServer(svc)
	defer target.Close()
	handler := ws.NewHandler(pubsub, mglog.NewMock(), authn, clients, channels)
	ts, err := newProxyHTPPServer(handler, target)
	require.Nil(t, err)
	defer ts.Close()
	pubsub.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	pubsub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cases := []struct {
		desc      string
		chanID    string
		subtopic  string
		header    bool
		clientKey string
		status    int
		err       error
		msg       []byte
	}{
		{
			desc:      "connect and send message",
			chanID:    id,
			subtopic:  "",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message with clientKey as query parameter",
			chanID:    id,
			subtopic:  "",
			header:    false,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message that cannot be published",
			chanID:    id,
			subtopic:  "",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       []byte{},
		},
		{
			desc:      "connect and send message to subtopic",
			chanID:    id,
			subtopic:  "subtopic",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message to nested subtopic",
			chanID:    id,
			subtopic:  "subtopic/nested",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect and send message to all subtopics",
			chanID:    id,
			subtopic:  ">",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusSwitchingProtocols,
			msg:       msg,
		},
		{
			desc:      "connect to empty channel",
			chanID:    "",
			subtopic:  "",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusBadGateway,
			msg:       []byte{},
		},
		{
			desc:      "connect with empty clientKey",
			chanID:    id,
			subtopic:  "",
			header:    true,
			clientKey: "",
			status:    http.StatusUnauthorized,
			msg:       []byte{},
		},
		{
			desc:      "connect and send message to subtopic with invalid name",
			chanID:    id,
			subtopic:  "sub/a*b/topic",
			header:    true,
			clientKey: clientKey,
			status:    http.StatusBadGateway,
			msg:       msg,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			conn, res, err := handshake(ts.URL, tc.chanID, tc.subtopic, tc.clientKey, tc.header)
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code '%d' got '%d'\n", tc.desc, tc.status, res.StatusCode))

			if tc.status == http.StatusSwitchingProtocols {
				assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))

				err = conn.WriteMessage(websocket.TextMessage, tc.msg)
				assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))
			}
		})
	}
}

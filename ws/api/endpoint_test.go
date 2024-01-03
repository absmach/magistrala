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

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging/mocks"
	"github.com/absmach/magistrala/ws"
	"github.com/absmach/magistrala/ws/api"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/absmach/mproxy/pkg/websockets"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	chanID     = "30315311-56ba-484d-b500-c1e08305511f"
	id         = "1"
	thingKey   = "c02ff576-ccd5-40f6-ba5f-c85377aad529"
	protocol   = "ws"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
)

var msg = []byte(`[{"n":"current","t":-1,"v":1.6}]`)

func newService(auth magistrala.AuthzServiceClient) (ws.Service, *mocks.PubSub) {
	pubsub := new(mocks.PubSub)
	return ws.New(auth, pubsub), pubsub
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

func makeURL(tsURL, chanID, subtopic, thingKey string, header bool) (string, error) {
	u, _ := url.Parse(tsURL)
	u.Scheme = protocol

	if chanID == "0" || chanID == "" {
		if header {
			return fmt.Sprintf("%s/channels/%s/messages", u, chanID), fmt.Errorf("invalid channel id")
		}
		return fmt.Sprintf("%s/channels/%s/messages?authorization=%s", u, chanID, thingKey), fmt.Errorf("invalid channel id")
	}

	subtopicPart := ""
	if subtopic != "" {
		subtopicPart = fmt.Sprintf("/%s", subtopic)
	}
	if header {
		return fmt.Sprintf("%s/channels/%s/messages%s", u, chanID, subtopicPart), nil
	}

	return fmt.Sprintf("%s/channels/%s/messages%s?authorization=%s", u, chanID, subtopicPart, thingKey), nil
}

func handshake(tsURL, chanID, subtopic, thingKey string, addHeader bool) (*websocket.Conn, *http.Response, error) {
	header := http.Header{}
	if addHeader {
		header.Add("Authorization", thingKey)
	}

	turl, _ := makeURL(tsURL, chanID, subtopic, thingKey, addHeader)
	conn, res, errRet := websocket.DefaultDialer.Dial(turl, header)

	return conn, res, errRet
}

func TestHandshake(t *testing.T) {
	auth := new(authmocks.Service)
	svc, pubsub := newService(auth)
	target := newHTTPServer(svc)
	defer target.Close()
	handler := ws.NewHandler(pubsub, mglog.NewMock(), auth)
	ts, err := newProxyHTPPServer(handler, target)
	require.Nil(t, err)
	defer ts.Close()
	auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{Subject: thingKey, Object: id, Domain: "", SubjectType: "thing", Permission: "publish", ObjectType: "group"}).Return(&magistrala.AuthorizeRes{Authorized: true, Id: "1"}, nil)
	auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{Subject: thingKey, Object: id, Domain: "", SubjectType: "thing", Permission: "subscribe", ObjectType: "group"}).Return(&magistrala.AuthorizeRes{Authorized: true, Id: "2"}, nil)
	auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false, Id: "3"}, nil)
	pubsub.On("Subscribe", mock.Anything, mock.Anything).Return(nil)
	pubsub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	cases := []struct {
		desc     string
		chanID   string
		subtopic string
		header   bool
		thingKey string
		status   int
		err      error
		msg      []byte
	}{
		{
			desc:     "connect and send message",
			chanID:   id,
			subtopic: "",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message with thingKey as query parameter",
			chanID:   id,
			subtopic: "",
			header:   false,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message that cannot be published",
			chanID:   id,
			subtopic: "",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic",
			chanID:   id,
			subtopic: "subtopic",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message to nested subtopic",
			chanID:   id,
			subtopic: "subtopic/nested",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect and send message to all subtopics",
			chanID:   id,
			subtopic: ">",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusSwitchingProtocols,
			msg:      msg,
		},
		{
			desc:     "connect to empty channel",
			chanID:   "",
			subtopic: "",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusBadGateway,
			msg:      []byte{},
		},
		{
			desc:     "connect with empty thingKey",
			chanID:   id,
			subtopic: "",
			header:   true,
			thingKey: "",
			status:   http.StatusUnauthorized,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic with invalid name",
			chanID:   id,
			subtopic: "sub/a*b/topic",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusBadGateway,
			msg:      msg,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			conn, res, err := handshake(ts.URL, tc.chanID, tc.subtopic, tc.thingKey, tc.header)
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code '%d' got '%d'\n", tc.desc, tc.status, res.StatusCode))

			if tc.status == http.StatusSwitchingProtocols {
				assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))

				err = conn.WriteMessage(websocket.TextMessage, tc.msg)
				assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))
			}
		})
	}
}

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/api"
	"github.com/mainflux/mainflux/coap/mocks"
	httpmock "github.com/mainflux/mainflux/http/mocks"
	log "github.com/mainflux/mainflux/logger"
	gocoap "github.com/plgd-dev/go-coap/v2"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/plgd-dev/go-coap/v2/udp"
	"github.com/plgd-dev/go-coap/v2/udp/client"
)

const (
	chanID   = "1"
	id       = "1"
	thingKey = "thing_key"
	protocol = "coap"
)

var (
	msg = []byte(`[{"n":"current","t":-1,"v":1.6}]`)
	// token = []byte{1}
)

func newService(cc mainflux.ThingsServiceClient) (coap.Service, mocks.MockPubSub) {
	pubsub := mocks.NewPubSub()
	return coap.New(cc, pubsub), pubsub
}

func newHTTPServer(svc coap.Service) *httptest.Server {
	mux := api.MakeHTTPHandler()
	return httptest.NewServer(mux)
}

func newCoAPServer(svc coap.Service) mux.HandlerFunc {
	logger := log.NewMock()
	handler := api.MakeCoAPHandler(svc, logger)
	return handler
	// return mux.NewRouter(), handlerFunc
	// gocoap.ListenAndServe("udp", fmt.Sprintf(":%s", "5683"), handler)
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

func handshake(tsURL, chanID, subtopic, thingKey string, addHeader bool) (*client.ClientConn, string, error) {
	url, _ := makeURL(tsURL, chanID, subtopic, thingKey, addHeader)
	conn, err := udp.Dial("localhost:5688")
	return conn, url, err
}

func TestHandshake(t *testing.T) {
	thingsClient := httpmock.NewThingsClient(map[string]string{thingKey: chanID})
	svc, _ := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()
	handler := newCoAPServer(svc)
	gocoap.ListenAndServe("udp", fmt.Sprintf(":%s", "5683"), handler)

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
			status:   http.StatusBadRequest,
			msg:      []byte{},
		},
		{
			desc:     "connect with empty thingKey",
			chanID:   id,
			subtopic: "",
			header:   true,
			thingKey: "",
			status:   http.StatusForbidden,
			msg:      []byte{},
		},
		{
			desc:     "connect and send message to subtopic with invalid name",
			chanID:   id,
			subtopic: "sub/a*b/topic",
			header:   true,
			thingKey: thingKey,
			status:   http.StatusBadRequest,
			msg:      msg,
		},
	}

	for _, tc := range cases {
		conn, url, err := handshake(ts.URL, tc.chanID, tc.subtopic, tc.thingKey, tc.header)
		fmt.Println(err)
		if err != nil {
			continue
		}

		resp, err := conn.Get(context.Background(), url)
		fmt.Println(resp, err)
	}

	// for _, tc := range cases {
	// 	conn, err := handshake(ts.URL, tc.chanID, tc.subtopic, tc.thingKey, tc.header)
	// 	assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code '%d' got '%d'\n", tc.desc, tc.status, res.StatusCode))

	// 	if tc.status == http.StatusSwitchingProtocols {
	// 		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))

	// 		r := pool.NewMessage()
	// 		r.SetToken(token)
	// 		r.SetBody(bytes.NewReader(msg))

	// 		resp, err := conn.Get()

	// 		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error %s\n", tc.desc, err))
	// 	}
	// }

}

// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
)

type publishReq struct {
	Topic   string `json:"topic"`
	Payload []byte `json:"payload"`
	QoS     byte   `json:"qos"`
	Retain  bool   `json:"retain"`
}

func setupFluxMQ(secret string, expectedTopic ...string) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /publish", func(w http.ResponseWriter, r *http.Request) {
		password := r.Header.Get("X-FluxMQ-Password")
		if password == "" || password != secret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req publishReq
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Topic == "" {
			http.Error(w, "empty topic", http.StatusBadRequest)
			return
		}
		if len(expectedTopic) > 0 && req.Topic != expectedTopic[0] {
			http.Error(w, fmt.Sprintf("unexpected topic: %s", req.Topic), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"healthy"}`)
	})

	return httptest.NewServer(mux)
}

func TestSendMessage(t *testing.T) {
	clientSecret := "validSecret"

	cases := []struct {
		desc     string
		topic    string
		domainID string
		wantTopic string
		msg      string
		secret   string
		err      errors.SDKError
	}{
		{
			desc:     "publish message successfully",
			topic:    "channelID",
			domainID: "domainID",
			wantTopic: "m/domainID/c/channelID",
			msg:      `[{"n":"current","t":-1,"v":1.6}]`,
			secret:   clientSecret,
			err:      nil,
		},
		{
			desc:     "publish message with subtopic",
			topic:    "channelID.sub.topic",
			domainID: "domainID",
			wantTopic: "m/domainID/c/channelID/sub/topic",
			msg:      `[{"n":"current","t":-1,"v":1.6}]`,
			secret:   clientSecret,
			err:      nil,
		},
		{
			desc:     "publish message with invalid secret",
			topic:    "channelID",
			domainID: "domainID",
			wantTopic: "m/domainID/c/channelID",
			msg:      `[{"n":"current","t":-1,"v":1.6}]`,
			secret:   "invalid",
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.New(""), errors.New("")), http.StatusUnauthorized),
		},
		{
			desc:     "publish message with empty secret",
			topic:    "channelID",
			domainID: "domainID",
			wantTopic: "m/domainID/c/channelID",
			msg:      `[{"n":"current","t":-1,"v":1.6}]`,
			secret:   "",
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.New(""), errors.New("")), http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ts := setupFluxMQ(clientSecret, tc.wantTopic)
			defer ts.Close()

			sdkConf := sdk.Config{
				HTTPAdapterURL:  ts.URL,
				MsgContentType:  "application/senml+json",
				TLSVerification: false,
			}
			mgsdk := sdk.NewSDK(sdkConf)

			err := mgsdk.SendMessage(context.Background(), tc.domainID, tc.topic, tc.msg, tc.secret)
			if tc.err != nil {
				assert.NotNil(t, err, fmt.Sprintf("%s: expected error, got nil", tc.desc))
			} else {
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error: %v", tc.desc, err))
			}
		})
	}
}

func TestSetContentType(t *testing.T) {
	sdkConf := sdk.Config{
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}
	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc  string
		cType sdk.ContentType
		err   errors.SDKError
	}{
		{
			desc:  "set senml+json content type",
			cType: "application/senml+json",
			err:   nil,
		},
		{
			desc:  "set json content type",
			cType: "application/json",
			err:   nil,
		},
		{
			desc:  "set invalid content type",
			cType: "invalid",
			err:   errors.NewSDKError(apiutil.ErrUnsupportedContentType),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := mgsdk.SetContentType(tc.cType)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		})
	}
}

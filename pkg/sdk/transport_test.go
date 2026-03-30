// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransport verifies that IdleConnTimeout=90s keeps connections pooled for
// healthy servers, and that network errors (EOF, reset) surface as descriptive errors.
func TestTransport(t *testing.T) {
	cases := []struct {
		desc        string
		serverFunc  func(t *testing.T) (url string, cleanup func())
		ctxFunc     func() context.Context
		wantErr     bool
		errContains string
	}{
		{
			desc: "make request successfully with connection reuse",
			serverFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				var connCount atomic.Int32
				srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"clients":[{"id":"1","name":"test-client"}]}`))
				}))
				srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
					if state == http.StateNew {
						connCount.Add(1)
					}
				}
				srv.Start()
				return srv.URL, func() {
					srv.Close()
					assert.Equal(t, int32(1), connCount.Load(), "expected connections to be reused (keep-alives enabled)")
				}
			},
			wantErr: false,
		},
		{
			desc: "make request with server closing connection",
			serverFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				go func() {
					for {
						conn, err := ln.Accept()
						if err != nil {
							return
						}
						conn.Close()
					}
				}()
				return "http://" + ln.Addr().String(), func() { ln.Close() }
			},
			wantErr:     true,
			errContains: "request failed",
		},
		{
			desc: "make request with connection reset by peer",
			serverFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				go func() {
					for {
						conn, err := ln.Accept()
						if err != nil {
							return
						}

						tcpConn, ok := conn.(*net.TCPConn)
						if ok {
							_ = tcpConn.SetLinger(0)
						}
						conn.Close()
					}
				}()
				return "http://" + ln.Addr().String(), func() { ln.Close() }
			},
			wantErr:     true,
			errContains: "request failed",
		},
		{
			desc: "make request with unreachable server",
			serverFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				addr := ln.Addr().String()
				ln.Close()
				return "http://" + addr, func() {}
			},
			wantErr:     true,
			errContains: "request failed",
		},
		{
			desc: "make request with cancelled context",
			serverFunc: func(t *testing.T) (string, func()) {
				t.Helper()
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				return srv.URL, srv.Close
			},
			ctxFunc: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr:     true,
			errContains: "request failed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			url, cleanup := tc.serverFunc(t)
			defer cleanup()

			smqsdk := sdk.NewSDK(sdk.Config{ClientsURL: url})

			ctx := context.Background()
			if tc.ctxFunc != nil {
				ctx = tc.ctxFunc()
			}

			client := sdk.Client{Name: "test-client"}
			for i := 0; i < 2; i++ {
				_, err := smqsdk.CreateClients(ctx, []sdk.Client{client}, domainID, validToken)
				if tc.wantErr {
					require.Error(t, err)
					if tc.errContains != "" {
						assert.True(t, strings.Contains(err.Error(), tc.errContains),
							"expected error %q to contain %q", err.Error(), tc.errContains)
					}
					break
				}
				require.NoError(t, err)
			}
		})
	}
}

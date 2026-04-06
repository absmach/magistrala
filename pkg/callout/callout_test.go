// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package callout_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/callout"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	operation      = "test_operation"
	entityType     = "client"
	userID         = "user_id"
	domainID       = "domain_id"
	filePermission = 0o644
)

var req = callout.Request{
	BaseRequest: callout.BaseRequest{
		Operation:  operation,
		EntityType: entityType,
	},
	Payload: map[string]any{
		"sender": userID,
		"time":   time.Now().UTC(),
		"domain": domainID,
	},
}

func TestNewCallout(t *testing.T) {
	cases := []struct {
		desc       string
		withTLS    bool
		certPath   string
		keyPath    string
		caPath     string
		timeout    time.Duration
		method     string
		urls       []string
		operations []string
		err        error
	}{
		{
			desc:       "successful callout creation without TLS",
			withTLS:    false,
			timeout:    time.Second,
			method:     http.MethodPost,
			urls:       []string{"http://example.com"},
			operations: []string{},
		},
		{
			desc:       "successful callout creation with TLS",
			withTLS:    true,
			certPath:   "client.crt",
			keyPath:    "client.key",
			caPath:     "ca.crt",
			timeout:    time.Second,
			method:     http.MethodPost,
			urls:       []string{"http://example.com"},
			operations: []string{},
		},
		{
			desc:       "failed callout creation with invalid cert",
			withTLS:    true,
			certPath:   "invalid.crt",
			keyPath:    "invalid.key",
			caPath:     "invalid.ca",
			timeout:    time.Second,
			method:     http.MethodPost,
			urls:       []string{"http://example.com"},
			operations: []string{},
			err:        errors.New("failed to initialize http client: tls: failed to find any PEM data in certificate input"),
		},
		{
			desc:       "invalid method",
			withTLS:    false,
			timeout:    time.Second,
			method:     "INVALID-METHOD",
			urls:       []string{"http://example.com"},
			operations: []string{},
			err:        errors.New("unsupported auth callout method: INVALID-METHOD"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			switch tc.desc {
			case "successful callout creation with TLS":
				generateAndWriteCertificates(t, tc.caPath, tc.certPath, tc.keyPath)

				defer func() {
					os.Remove(tc.certPath)
					os.Remove(tc.keyPath)
					os.Remove(tc.caPath)
				}()
			case "failed callout creation with invalid cert":
				writeFile(t, tc.certPath, []byte("invalid cert content"))
				writeFile(t, tc.keyPath, []byte("invalid key content"))
				writeFile(t, tc.caPath, []byte("invalid ca content"))

				defer func() {
					os.Remove(tc.certPath)
					os.Remove(tc.keyPath)
					os.Remove(tc.caPath)
				}()
			}

			client, err := callout.New(callout.Config{
				TLSVerification: tc.withTLS,
				Cert:            tc.certPath,
				Key:             tc.keyPath,
				CACert:          tc.caPath,
				Timeout:         tc.timeout,
				Method:          tc.method,
				URLs:            tc.urls,
				Operations:      tc.operations,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.NotNil(t, client)
			}
		})
	}
}

func generateAndWriteCertificates(t *testing.T, caPath, certPath, keyPath string) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate CA private key")

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err, "Failed to create CA certificate")

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate client private key")

	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
			CommonName:   "Test Client",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	clientBytes, err := x509.CreateCertificate(rand.Reader, &clientTemplate, &caTemplate, &clientKey.PublicKey, caKey)
	require.NoError(t, err, "Failed to create client certificate")

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	writeFile(t, caPath, caPEM)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: clientBytes,
	})
	writeFile(t, certPath, certPEM)

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(clientKey),
	})
	writeFile(t, keyPath, keyPEM)
}

func writeFile(t *testing.T, path string, content []byte) {
	err := os.WriteFile(path, content, filePermission)
	require.NoError(t, err, "Failed to write file: %s", path)
}

func TestCallout_MakeRequest(t *testing.T) {
	cases := []struct {
		desc          string
		serverHandler http.HandlerFunc
		method        string
		contextSetup  func() context.Context
		urls          []string
		operations    []string
		expectError   bool
		err           error
	}{
		{
			desc: "successful POST request",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				w.WriteHeader(http.StatusOK)
			}),
			method:       http.MethodPost,
			contextSetup: func() context.Context { return context.Background() },
			operations:   []string{operation},
			expectError:  false,
		},
		{
			desc: "successful GET request with query params",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, domainID, r.URL.Query().Get("domain"))
				assert.Equal(t, userID, r.URL.Query().Get("sender"))
				w.WriteHeader(http.StatusOK)
			}),
			method:       http.MethodGet,
			contextSetup: func() context.Context { return context.Background() },
			operations:   []string{operation},
			expectError:  false,
		},
		{
			desc: "server returns forbidden status",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			}),
			method:       http.MethodPost,
			contextSetup: func() context.Context { return context.Background() },
			operations:   []string{operation},
			expectError:  true,
		},
		{
			desc:         "invalid URL",
			method:       http.MethodGet,
			contextSetup: func() context.Context { return context.Background() },
			urls:         []string{"http://invalid-url"},
			operations:   []string{operation},
			expectError:  true,
		},
		{
			desc: "cancelled context",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			method: http.MethodGet,
			contextSetup: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			operations:  []string{operation},
			expectError: true,
		},
		{
			desc: "multiple URLs all succeed",
			serverHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			method:       http.MethodPost,
			contextSetup: func() context.Context { return context.Background() },
			operations:   []string{operation},
			expectError:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var servers []*httptest.Server
			var urls []string

			if tc.desc == "invalid URL" {
				urls = tc.urls
			} else {
				if tc.desc == "multiple URLs all succeed" {
					// Create multiple test servers
					for i := 0; i < 2; i++ {
						server := httptest.NewServer(tc.serverHandler)
						servers = append(servers, server)
						urls = append(urls, server.URL)
					}
				} else {
					server := httptest.NewServer(tc.serverHandler)
					servers = append(servers, server)
					urls = append(urls, server.URL)
				}
			}

			defer func() {
				for _, server := range servers {
					server.Close()
				}
			}()

			// Create a callout with a short timeout for tests
			cb, err := callout.New(
				callout.Config{
					TLSVerification: false,
					Cert:            "",
					Key:             "",
					CACert:          "",
					Timeout:         time.Second,
					Method:          tc.method,
					URLs:            urls,
					Operations:      tc.operations,
				})
			assert.NoError(t, err)

			ctx := tc.contextSetup()
			err = cb.Callout(ctx, req)

			if tc.expectError {
				assert.Error(t, err)
				if tc.err != nil {
					assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCallout_Operations(t *testing.T) {
	cases := []struct {
		desc         string
		operations   []string
		request      callout.Request
		serverCalled bool
	}{
		{
			desc:         "matching operation is called",
			operations:   []string{operation},
			request:      req,
			serverCalled: true,
		},
		{
			desc:         "non-matching operation is not called",
			operations:   []string{"other_operation"},
			request:      req,
			serverCalled: false,
		},
		{
			desc:         "empty operations list calls always",
			operations:   []string{},
			request:      req,
			serverCalled: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			serverCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				serverCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			cb, err := callout.New(callout.Config{
				TLSVerification: false,
				Cert:            "",
				Key:             "",
				CACert:          "",
				Timeout:         time.Second,
				Method:          http.MethodPost,
				URLs:            []string{ts.URL},
				Operations:      tc.operations,
			})
			assert.NoError(t, err)

			err = cb.Callout(context.Background(), tc.request)
			assert.NoError(t, err)
			assert.Equal(t, tc.serverCalled, serverCalled, "Server call status does not match expected")
		})
	}
}

func TestCallout_NoURLs(t *testing.T) {
	cb, err := callout.New(callout.Config{
		TLSVerification: false,
		Cert:            "",
		Key:             "",
		CACert:          "",
		Timeout:         time.Second,
		Method:          http.MethodPost,
		URLs:            []string{},
		Operations:      []string{operation},
	})
	assert.NoError(t, err)

	err = cb.Callout(context.Background(), req)
	assert.NoError(t, err, "No error should be returned when URL list is empty")
}

// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/mainflux/mainflux"
	bsmocks "github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/certs/mocks"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	thmocks "github.com/mainflux/mainflux/things/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongValue = "wrong-value"
	email      = "user@example.com"
	token      = "token"
	thingsNum  = 1
	name       = "certificate name"
	thingKey   = "thingKey"
	thingID    = "1"

	ttl               = "1h"
	caPath            = "../docker/ssl/certs/ca.crt"
	caKeyPath         = "../docker/ssl/certs/ca.key"
	cfgSignHoursValid = "24h"
	cfgSignRSABits    = 2048
)

func newService(tokens map[string]string) (certs.Service, error) {
	ac := bsmocks.NewAuthClient(map[string]string{token: email})
	server := newThingsServer(newThingsService(ac))

	policies := []thmocks.MockSubjectSet{{Object: "users", Relation: "member"}}
	auth := thmocks.NewAuthService(tokens, map[string][]thmocks.MockSubjectSet{email: policies})
	config := mfsdk.Config{
		ThingsURL: server.URL,
	}

	sdk := mfsdk.NewSDK(config)
	repo := mocks.NewCertsRepository()

	tlsCert, caCert, err := loadCertificates(caPath, caKeyPath)
	if err != nil {
		return nil, err
	}

	idp := uuid.NewMock()
	pki := mocks.NewPkiAgent(tlsCert, caCert, cfgSignRSABits, cfgSignHoursValid)

	return certs.New(auth, repo, idp, pki, sdk), nil
}

func newThingsService(auth mainflux.AuthServiceClient) things.Service {
	ths := make(map[string]things.Thing, thingsNum)
	for i := 0; i < thingsNum; i++ {
		id := strconv.Itoa(i + 1)
		ths[id] = things.Thing{
			ID:    id,
			Key:   thingKey,
			Owner: email,
		}
	}

	return bsmocks.NewThingsService(ths, map[string]things.Channel{}, auth)
}

func TestIssueCert(t *testing.T) {
	svc, err := newService(map[string]string{token: email})
	require.Nil(t, err, fmt.Sprintf("unexpected service creation error: %s\n", err))

	cases := []struct {
		name    string
		token   string
		desc    string
		thingID string
		ttl     string
		key     string
		err     error
	}{
		{
			desc:    "issue new cert",
			token:   token,
			name:    name,
			thingID: thingID,
			ttl:     ttl,
			err:     nil,
		},
		{
			desc:    "issue new cert for non existing thing id",
			name:    name,
			token:   wrongValue,
			thingID: thingID,
			ttl:     ttl,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "issue new cert for non existing thing id",
			name:    name,
			token:   token,
			thingID: "2",
			ttl:     ttl,
			err:     certs.ErrThingRetrieve,
		},
	}

	for _, tc := range cases {
		c, err := svc.IssueCert(context.Background(), tc.token, tc.thingID, tc.name, tc.ttl)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		cert, _ := readCert([]byte(c.Certificate))
		if cert != nil {
			assert.True(t, strings.Contains(cert.Subject.CommonName, thingKey), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		}
	}

}

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func loadCertificates(caPath, caKeyPath string) (tls.Certificate, *x509.Certificate, error) {
	var tlsCert tls.Certificate
	var caCert *x509.Certificate

	if caPath == "" || caKeyPath == "" {
		return tlsCert, caCert, nil
	}

	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return tlsCert, caCert, err
	}

	if _, err := os.Stat(caKeyPath); os.IsNotExist(err) {
		return tlsCert, caCert, err
	}

	tlsCert, err := tls.LoadX509KeyPair(caPath, caKeyPath)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(err, err)
	}

	b, err := os.ReadFile(caPath)
	if err != nil {
		return tlsCert, caCert, err
	}

	caCert, err = readCert(b)
	if err != nil {
		return tlsCert, caCert, errors.Wrap(err, err)
	}

	return tlsCert, caCert, nil
}

func readCert(b []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("failed to decode PEM data")
	}

	return x509.ParseCertificate(block.Bytes)
}

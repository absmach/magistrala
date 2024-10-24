// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var cert = mgsdk.Cert{
	ClientID: client.ID,
}

func TestGetCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	var ct mgsdk.Cert
	var cts mgsdk.CertSerials

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		serials       mgsdk.CertSerials
		cert          mgsdk.Cert
	}{
		{
			desc: "get cert successfully",
			args: []string{
				"client",
				client.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			serials: mgsdk.CertSerials{
				PageRes: mgsdk.PageRes{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Certs: []mgsdk.Cert{cert},
			},
		},
		{
			desc: "get cert successfully by id",
			args: []string{
				client.ID,
				domainID,
				validToken,
			},
			logType: entityLog,
			cert:    cert,
		},
		{
			desc: "get cert with invalid token",
			args: []string{
				"client",
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "get cert by id with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusUnauthorized)),
			logType:       errLog,
		},
		{
			desc: "get cert with invalid args",
			args: []string{
				client.ID,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ViewCertByClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.serials, tc.sdkErr)
			sdkCall1 := sdkMock.On("ViewCert", mock.Anything, mock.Anything, mock.Anything).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{getCmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				if tc.args[1] == "client" {
					err := json.Unmarshal([]byte(out), &cts)
					assert.Nil(t, err)
					assert.Equal(t, tc.serials, cts, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.serials, cts))
				} else {
					err := json.Unmarshal([]byte(out), &ct)
					assert.Nil(t, err)
					assert.Equal(t, tc.cert, ct, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, ct))
				}

			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
			sdkCall1.Unset()
		})
	}
}

func TestRevokeCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	revokeTime := time.Now()

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		logType       outputLog
		errLogMessage string
		time          time.Time
		response      string
	}{
		{
			desc: "revoke cert successfully",
			args: []string{
				client.ID,
				domainID,
				token,
			},
			logType:  revokeLog,
			response: fmt.Sprintf("\nrevoked: %s\n\n", revokeTime),
			time:     revokeTime,
		},
		{
			desc: "revoke cert with invalid args",
			args: []string{
				client.ID,
				domainID,
				token,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "revoke cert with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RevokeCert", tc.args[0], tc.args[1], tc.args[2]).Return(tc.time, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{revokeCmd}, tc.args...)...)

			switch tc.logType {
			case revokeLog:
				assert.Equal(t, tc.response, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.response, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestIssueCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cert := mgsdk.Cert{
		SerialNumber: "serial",
	}

	var cs mgsdk.Cert
	cases := []struct {
		desc          string
		args          []string
		logType       outputLog
		errLogMessage string
		sdkErr        errors.SDKError
		cert          mgsdk.Cert
	}{
		{
			desc: "issue cert successfully",
			args: []string{
				client.ID,
				domainID,
				validToken,
			},
			cert:    cert,
			logType: entityLog,
		},
		{
			desc: "issue cert with invalid args",
			args: []string{
				client.ID,
				domainID,
				validToken,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "issue cert with invalid token",
			args: []string{
				client.ID,
				domainID,
				invalidToken,
			},
			logType:       errLog,
			sdkErr:        errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden)),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("IssueCert", mock.Anything, mock.Anything, tc.args[1], tc.args[2]).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{issueCmd}, tc.args...)...)

			switch tc.logType {
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &cs)
				assert.Nil(t, err)
				assert.Equal(t, tc.cert, cs, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, cs))
			}
			sdkCall.Unset()
		})
	}
}

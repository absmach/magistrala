// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/sdk"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	revokeCmd      = "revoke"
	deleteCmd      = "delete"
	issueCmd       = "issue"
	renewCmd       = "renew"
	certsListCmd   = "get"
	downloadCACmd  = "download-ca"
	CATokenCmd     = "certsToken-ca"
	viewCACmd      = "view-ca"
	filePermission = 0o644
)

var (
	serialNumber  = "39054620502613157373429341617471746606"
	id            = "5b4c9ee3-e719-4a0a-9ee5-354932c5e6a4"
	commonName    = "test-name"
	certsToken    = "certsToken"
	certsDomainID = "domain-id"
)

func TestIssueCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	ipAddrs := "[\"192.168.100.22\"]"

	var cert sdk.Certificate
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		cert          sdk.Certificate
	}{
		{
			desc: "issue cert successfully",
			args: []string{
				id,
				commonName,
				ipAddrs,
				certsDomainID,
				certsToken,
			},
			logType: entityLog,
			cert:    sdk.Certificate{SerialNumber: serialNumber},
		},
		{
			desc: "issue cert with invalid args",
			args: []string{
				id,
				ipAddrs,
			},
			logType: usageLog,
		},
		{
			desc: "issue cert failed",
			args: []string{
				id,
				commonName,
				ipAddrs,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrCreateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrCreateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "issue cert with 6 args",
			args: []string{
				id,
				commonName,
				ipAddrs,
				"{\"organization\":[\"organization_name\"]}",
				certsDomainID,
				certsToken,
			},
			logType: entityLog,
			cert:    sdk.Certificate{SerialNumber: serialNumber},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			defer func() {
				cleanupFiles(t, []string{"cert.pem", "key.pem"})
			}()
			sdkCall := sdkMock.On("IssueCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{issueCmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				lines := strings.Split(out, "\n")
				var jsonLines []string
				var inJSON bool

				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "{") {
						inJSON = true
						jsonLines = append(jsonLines, line)
					} else if inJSON && strings.HasSuffix(line, "}") {
						jsonLines = append(jsonLines, line)
						break
					} else if inJSON {
						jsonLines = append(jsonLines, line)
					}
				}

				if len(jsonLines) == 0 {
					t.Fatalf("No JSON found in output: %s", out)
				}

				jsonPart := strings.Join(jsonLines, "")

				err := json.Unmarshal([]byte(jsonPart), &cert)
				assert.Nil(t, err)
				assert.Equal(t, tc.cert, cert, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, cert))
				assert.True(t, strings.Contains(out, "All certificate files have been saved successfully"), fmt.Sprintf("%s should save files", tc.desc))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestRevokeCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "revoke cert successfully",
			args: []string{
				serialNumber,
				certsDomainID,
				certsToken,
			},
			logType: okLog,
		},
		{
			desc: "revoke cert with invalid args",
			args: []string{
				serialNumber,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "revoke cert failed",
			args: []string{
				serialNumber,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RevokeCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{revokeCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestDeleteCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "delete certs successfully",
			args: []string{
				id,
				certsDomainID,
				certsToken,
			},
			logType: okLog,
		},
		{
			desc: "delete certs with invalid args",
			args: []string{
				id,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "delete certs failed",
			args: []string{
				id,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("DeleteCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{deleteCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestRenewCertCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
	}{
		{
			desc: "renew cert successfully",
			args: []string{
				serialNumber,
				certsDomainID,
				certsToken,
			},
			logType: okLog,
		},
		{
			desc: "renew cert with invalid args",
			args: []string{
				serialNumber,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "renew cert failed",
			args: []string{
				serialNumber,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("RenewCert", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(sdk.Certificate{}, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{renewCmd}, tc.args...)...)
			switch tc.logType {
			case okLog:
				assert.True(t, strings.Contains(out, "ok"), fmt.Sprintf("%s unexpected response: expected success message, got: %v", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestListCertsCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	var page sdk.CertificatePage
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		page          sdk.CertificatePage
	}{
		{
			desc: "list certs successfully",
			args: []string{
				all,
				certsDomainID,
				certsToken,
			},
			logType: entityLog,
			page: sdk.CertificatePage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Certificates: []sdk.Certificate{
					{SerialNumber: serialNumber},
				},
			},
		},
		{
			desc: "list certs successfully with entity ID",
			args: []string{
				id,
				certsDomainID,
				certsToken,
			},
			logType: entityLog,
			page: sdk.CertificatePage{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Certificates: []sdk.Certificate{
					{SerialNumber: serialNumber},
				},
			},
		},
		{
			desc: "list certs with invalid args",
			args: []string{
				all,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "failed list certs with all",
			args: []string{
				all,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "failed list certs with entity ID",
			args: []string{
				id,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ListCerts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{certsListCmd}, tc.args...)...)

			switch tc.logType {
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case entityLog:
				err := json.Unmarshal([]byte(out), &page)
				if err != nil {
					t.Fatalf("Failed to unmarshal JSON: %v", err)
				}
				assert.Equal(t, tc.page, page, fmt.Sprintf("%v unexpected response, expected: %v, got: %v", tc.desc, tc.page, page))
			}

			sdkCall.Unset()
		})
	}
}

func TestDownloadCACmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logMessage    string
		logType       outputLog
		certBundle    sdk.CertificateBundle
	}{
		{
			desc:    "download CA successfully",
			args:    []string{},
			logType: entityLog,
			certBundle: sdk.CertificateBundle{
				Certificate: []byte("certificate"),
			},
			logMessage: "Saved ca.crt\n\nAll certificate files have been saved successfully.\n",
		},
		{
			desc: "download CA with invalid args",
			args: []string{
				extraArg,
			},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			defer func() {
				cleanupFiles(t, []string{"ca.crt"})
			}()
			sdkCall := sdkMock.On("DownloadCA", mock.Anything).Return(tc.certBundle, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{downloadCACmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				assert.True(t, strings.Contains(out, "Saved ca.crt"), fmt.Sprintf("%s invalid output: %s", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestViewCACmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	var cert sdk.Certificate
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		cert          sdk.Certificate
	}{
		{
			desc:    "view cert successfully",
			args:    []string{},
			logType: entityLog,
			cert: sdk.Certificate{
				Certificate: "certificate",
				Key:         "privatekey",
			},
		},
		{
			desc:          "view cert failed",
			args:          []string{},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrUpdateEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
			cert:          sdk.Certificate{},
		},
		{
			desc:    "view cert with invalid args",
			args:    []string{extraArg},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("ViewCA", mock.Anything).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{viewCACmd}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				err := json.Unmarshal([]byte(out), &cert)
				assert.Nil(t, err)
				assert.Equal(t, tc.cert, cert, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, cert))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestGenerateCRLCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		crlBytes      []byte
	}{
		{
			desc:     "generate CRL successfully",
			args:     []string{},
			logType:  entityLog,
			crlBytes: []byte("crl-data"),
		},
		{
			desc:          "generate CRL failed",
			args:          []string{},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc:    "generate CRL with invalid args",
			args:    []string{"invalid"},
			logType: usageLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			defer func() {
				cleanupFiles(t, []string{"ca.crl"})
			}()

			sdkCall := sdkMock.On("GenerateCRL", mock.Anything).Return(tc.crlBytes, tc.sdkErr)
			defer sdkCall.Unset()

			out := executeCommand(t, rootCmd, append([]string{"crl"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				assert.True(t, strings.Contains(out, "CRL file has been saved successfully"), fmt.Sprintf("%s invalid output: %s", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
		})
	}
}

func TestGetEntityIDCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	entityID := "test-entity-id"

	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		entityID      string
	}{
		{
			desc:     "get entity ID successfully",
			args:     []string{serialNumber, certsDomainID, certsToken},
			logType:  entityLog,
			entityID: entityID,
		},
		{
			desc:    "get entity ID with invalid args",
			args:    []string{serialNumber, extraArg},
			logType: usageLog,
		},
		{
			desc:          "get entity ID failed",
			args:          []string{serialNumber, certsDomainID, certsToken},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrViewEntity, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			sdkCall := sdkMock.On("EntityID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.entityID, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"entity-id"}, tc.args...)...)

			switch tc.logType {
			case entityLog:
				assert.True(t, strings.Contains(out, tc.entityID), fmt.Sprintf("%s invalid output: %s", tc.desc, out))
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			case errLog:
				assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
			}
			sdkCall.Unset()
		})
	}
}

func cleanupFiles(t *testing.T, filenames []string) {
	for _, filename := range filenames {
		err := os.Remove(filename)
		if err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove file %s: %v", filename, err)
		}
	}
}

func TestIssueFromCSRInternalCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	agentToken := "agent-certsToken-123"
	csrPath := "test.csr"
	bytes := []byte("-----BEGIN CERTIFICATE REQUEST-----\n-csr-content\n-----END CERTIFICATE REQUEST-----")

	err := os.WriteFile(csrPath, bytes, filePermission)
	if err != nil {
		t.Fatalf("Failed to create test CSR file: %v", err)
	}
	defer os.Remove(csrPath)

	var cert sdk.Certificate
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		cert          sdk.Certificate
	}{
		{
			desc: "issue cert from CSR internal successfully",
			args: []string{
				id,
				"10h",
				csrPath,
				agentToken,
			},
			logType: entityLog,
			cert:    sdk.Certificate{SerialNumber: serialNumber},
		},
		{
			desc: "issue cert from CSR internal with invalid args",
			args: []string{
				id,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "issue cert from CSR internal failed",
			args: []string{
				id,
				"10h",
				csrPath,
				agentToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "issue cert from CSR internal with non-existent file",
			args: []string{
				id,
				"10h",
				"non-existent.csr",
				agentToken,
			},
			logType: errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			defer func() {
				cleanupFiles(t, []string{"cert.pem", "key.pem"})
			}()
			sdkCall := sdkMock.On("IssueFromCSRInternal", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"issue-csr-internal"}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				lines := strings.Split(out, "\n")
				var jsonLines []string
				var inJSON bool

				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "{") {
						inJSON = true
						jsonLines = append(jsonLines, line)
					} else if inJSON && strings.HasSuffix(line, "}") {
						jsonLines = append(jsonLines, line)
						break
					} else if inJSON {
						jsonLines = append(jsonLines, line)
					}
				}

				if len(jsonLines) == 0 {
					t.Fatalf("No JSON found in output: %s", out)
				}

				jsonPart := strings.Join(jsonLines, "")

				err := json.Unmarshal([]byte(jsonPart), &cert)
				assert.Nil(t, err)
				assert.Equal(t, tc.cert, cert, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, cert))
				assert.True(t, strings.Contains(out, "All certificate files have been saved successfully"), fmt.Sprintf("%s should save files", tc.desc))
			case errLog:
				if tc.errLogMessage != "" {
					assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
				} else {
					assert.True(t, strings.Contains(out, "error"), fmt.Sprintf("%s should contain error message: %s", tc.desc, out))
				}
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}

func TestIssueFromCSRCmd(t *testing.T) {
	sdkMock := new(sdkmocks.SDK)
	cli.SetSDK(sdkMock)
	certCmd := cli.NewCertsCmd()
	rootCmd := setFlags(certCmd)

	csrPath := "test.csr"
	bytes := []byte("-----BEGIN CERTIFICATE REQUEST-----\n-csr-content\n-----END CERTIFICATE REQUEST-----")

	err := os.WriteFile(csrPath, bytes, filePermission)
	if err != nil {
		t.Fatalf("Failed to create test CSR file: %v", err)
	}
	defer os.Remove(csrPath)

	var cert sdk.Certificate
	cases := []struct {
		desc          string
		args          []string
		sdkErr        errors.SDKError
		errLogMessage string
		logType       outputLog
		cert          sdk.Certificate
	}{
		{
			desc: "issue cert from CSR successfully",
			args: []string{
				id,
				"10h",
				csrPath,
				certsDomainID,
				certsToken,
			},
			logType: entityLog,
			cert:    sdk.Certificate{SerialNumber: serialNumber},
		},
		{
			desc: "issue cert from CSR with invalid args",
			args: []string{
				id,
				extraArg,
			},
			logType: usageLog,
		},
		{
			desc: "issue cert from CSR failed",
			args: []string{
				id,
				"10h",
				csrPath,
				certsDomainID,
				certsToken,
			},
			sdkErr:        errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity),
			errLogMessage: fmt.Sprintf("\nerror: %s\n\n", errors.NewSDKErrorWithStatus(certs.ErrFailedCertCreation, http.StatusUnprocessableEntity)),
			logType:       errLog,
		},
		{
			desc: "issue cert from CSR with non-existent file",
			args: []string{
				id,
				"10h",
				"non-existent.csr",
				certsDomainID,
				certsToken,
			},
			logType: errLog,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			defer func() {
				cleanupFiles(t, []string{"cert.pem", "key.pem"})
			}()
			sdkCall := sdkMock.On("IssueFromCSR", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.cert, tc.sdkErr)
			out := executeCommand(t, rootCmd, append([]string{"issue-csr"}, tc.args...)...)
			switch tc.logType {
			case entityLog:
				lines := strings.Split(out, "\n")
				var jsonLines []string
				var inJSON bool

				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "{") {
						inJSON = true
						jsonLines = append(jsonLines, line)
					} else if inJSON && strings.HasSuffix(line, "}") {
						jsonLines = append(jsonLines, line)
						break
					} else if inJSON {
						jsonLines = append(jsonLines, line)
					}
				}

				if len(jsonLines) == 0 {
					t.Fatalf("No JSON found in output: %s", out)
				}

				jsonPart := strings.Join(jsonLines, "")

				err := json.Unmarshal([]byte(jsonPart), &cert)
				assert.Nil(t, err)
				assert.Equal(t, tc.cert, cert, fmt.Sprintf("%s unexpected response: expected: %v, got: %v", tc.desc, tc.cert, cert))
				assert.True(t, strings.Contains(out, "All certificate files have been saved successfully"), fmt.Sprintf("%s should save files", tc.desc))
			case errLog:
				if tc.errLogMessage != "" {
					assert.Equal(t, tc.errLogMessage, out, fmt.Sprintf("%s unexpected error response: expected %s got errLogMessage:%s", tc.desc, tc.errLogMessage, out))
				} else {
					assert.True(t, strings.Contains(out, "error"), fmt.Sprintf("%s should contain error message: %s", tc.desc, out))
				}
			case usageLog:
				assert.False(t, strings.Contains(out, rootCmd.Use), fmt.Sprintf("%s invalid usage: %s", tc.desc, out))
			}
			sdkCall.Unset()
		})
	}
}
